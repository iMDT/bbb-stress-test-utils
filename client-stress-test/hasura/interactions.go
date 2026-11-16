package hasura

import (
	"bbb-stress-test/common"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Deterministic interactions (raise hand / reactions) so every run produces the
// same load pattern. Targets are always the users at the END of the user list,
// which forces the moderator's paginated user list to keep moving.
//
// All simulated users live in this process, so a global coordinator picks the
// actors and sends the mutations through their own connections.

var (
	joinedUsersMu sync.Mutex
	joinedUsers   []*common.User

	// FIFO of users with hand raised by the coordinator (oldest first).
	raisedHandsMu sync.Mutex
	raisedHands   []*common.User
)

func RegisterJoinedUser(user *common.User) {
	joinedUsersMu.Lock()
	defer joinedUsersMu.Unlock()
	for _, u := range joinedUsers {
		if u == user {
			return
		}
	}
	joinedUsers = append(joinedUsers, user)
}

func isActive(u *common.User) bool { return u.Joined && !u.WsConnectionClosed }

// isViewer uses the live role from user_current, so users promoted by
// moderatorActions are excluded too.
func isViewer(u *common.User) bool { return !u.IsModerator && !u.CurrentIsModerator }

// usersInUserListOrder returns the active simulated users sorted like the
// client's Patched_UserListSubscription:
// presenter desc, role asc (MODERATOR < VIEWER), whiteboardWriteAccess desc,
// nameSortable asc, registeredAt asc, userId asc.
func usersInUserListOrder() []*common.User {
	joinedUsersMu.Lock()
	var list []*common.User
	for _, u := range joinedUsers {
		if isActive(u) {
			list = append(list, u)
		}
	}
	joinedUsersMu.Unlock()

	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Presenter != b.Presenter {
			return a.Presenter
		}
		am, bm := b2i(a.IsModerator || a.CurrentIsModerator), b2i(b.IsModerator || b.CurrentIsModerator)
		if am != bm {
			return am > bm
		}
		if a.WhiteboardWriteAccess != b.WhiteboardWriteAccess {
			return a.WhiteboardWriteAccess
		}
		an, bn := a.NameSortable, b.NameSortable
		if an == "" {
			an = a.Name
		}
		if bn == "" {
			bn = b.Name
		}
		if an != bn {
			return an < bn
		}
		if a.RegisteredAt != b.RegisteredAt {
			return a.RegisteredAt < b.RegisteredAt
		}
		return a.UserId < b.UserId
	})
	return list
}

func findModerator() *common.User {
	joinedUsersMu.Lock()
	defer joinedUsersMu.Unlock()
	for _, u := range joinedUsers {
		if u.IsModerator && isActive(u) {
			return u
		}
	}
	return nil
}

// StartInteractionCoordinators launches the loops enabled in the config.
// Call once from main.
func StartInteractionCoordinators() {
	cfg := common.GetConfig()
	if cfg.RaiseHand.Enabled {
		go runRaiseHandLoop(cfg.RaiseHand)
		go runLowerHandLoop(cfg.RaiseHand)
	}
	if cfg.Reactions.Enabled {
		go runReactionsLoop(cfg.Reactions)
	}
}

// runRaiseHandLoop: every RaiseIntervalSecs, the last viewer in the user list
// whose hand is down raises it.
func runRaiseHandLoop(cfg common.RaiseHandConfig) {
	logger := log.WithField("_routine", "raise-hand")
	interval := time.Duration(orDefault(cfg.RaiseIntervalSecs, 30)) * time.Second

	for {
		time.Sleep(interval)
		list := usersInUserListOrder()

		var target *common.User
		for i := len(list) - 1; i >= 0; i-- {
			if isViewer(list[i]) && !list[i].HandRaised {
				target = list[i]
				break
			}
		}
		if target == nil {
			logger.Info("No viewer available to raise hand")
			continue
		}

		target.HandRaised = true
		raisedHandsMu.Lock()
		raisedHands = append(raisedHands, target)
		pending := len(raisedHands)
		raisedHandsMu.Unlock()

		logger.Infof("%s raises hand (%d hands raised)", target.Name, pending)
		SendSetRaiseHand(target, true, "")
	}
}

// runLowerHandLoop: every LowerIntervalSecs, the moderator lowers the hand that
// has been raised the longest.
func runLowerHandLoop(cfg common.RaiseHandConfig) {
	logger := log.WithField("_routine", "raise-hand")
	interval := time.Duration(orDefault(cfg.LowerIntervalSecs, 60)) * time.Second

	for {
		time.Sleep(interval)

		raisedHandsMu.Lock()
		if len(raisedHands) == 0 {
			raisedHandsMu.Unlock()
			logger.Info("No raised hand to lower")
			continue
		}
		target := raisedHands[0]
		raisedHands = raisedHands[1:]
		pending := len(raisedHands)
		raisedHandsMu.Unlock()

		if mod := findModerator(); mod != nil {
			logger.Infof("Moderator %s lowers hand of %s (%d hands still raised)", mod.Name, target.Name, pending)
			SendSetRaiseHand(mod, false, target.UserId)
		} else {
			logger.Infof("No moderator available, %s lowers own hand (%d hands still raised)", target.Name, pending)
			SendSetRaiseHand(target, false, "")
		}
		target.HandRaised = false
	}
}

// runReactionsLoop: every IntervalSecs, the users at the end of the user list
// react. Groups are applied from the end of the list backwards: with groups
// [👏 10%, 😃 10%, 👍 10%] the last 10% clap, the 10% before them smile, and so
// on. Groups are sent DelayBetweenGroupsMs apart.
func runReactionsLoop(cfg common.ReactionsConfig) {
	logger := log.WithField("_routine", "reactions")
	interval := time.Duration(orDefault(cfg.IntervalSecs, 60)) * time.Second
	groupDelay := time.Duration(orDefault(cfg.DelayBetweenGroupsMs, 3000)) * time.Millisecond
	groups := cfg.Groups
	if len(groups) == 0 {
		groups = []common.ReactionGroup{
			{Emoji: "👏", Percent: 10},
			{Emoji: "😃", Percent: 10},
			{Emoji: "👍", Percent: 10},
		}
	}

	for {
		time.Sleep(interval)
		list := usersInUserListOrder()
		total := len(list)
		if total == 0 {
			continue
		}

		end := total // exclusive index into list; walk backwards
		for gi, g := range groups {
			n := total * g.Percent / 100
			if n > end {
				n = end
			}
			members := list[end-n : end]
			end -= n

			logger.Infof("%d of %d users react with %s", len(members), total, g.Emoji)
			for _, u := range members {
				SendSetReactionEmoji(u, g.Emoji)
			}

			if gi < len(groups)-1 {
				time.Sleep(groupDelay)
			}
		}
	}
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// SendSetRaiseHand raises/lowers a hand. With an empty targetUserId it acts on
// the sender; moderators can pass another user's ID to lower their hand.
func SendSetRaiseHand(user *common.User, raise bool, targetUserId string) {
	if user.WsConnectionClosed {
		return
	}
	vars := map[string]interface{}{"raiseHand": raise}
	query := `mutation SetRaiseHand($raiseHand: Boolean!) {
			userSetRaiseHand(raiseHand: $raiseHand)
		}`
	if targetUserId != "" {
		vars["userId"] = targetUserId
		query = `mutation SetRaiseHand($userId: String!, $raiseHand: Boolean!) {
			userSetRaiseHand(userId: $userId, raiseHand: $raiseHand)
		}`
	}
	SendGenericGraphqlMessage(user, GetCurrMessageId(user), vars, "SetRaiseHand", query)
}

// SendSetReactionEmoji sends a reaction (shown over the user's avatar).
func SendSetReactionEmoji(user *common.User, emoji string) {
	if user.WsConnectionClosed {
		return
	}
	SendGenericGraphqlMessage(
		user,
		GetCurrMessageId(user),
		map[string]interface{}{"reactionEmoji": emoji},
		"SetReactionEmoji",
		`mutation SetReactionEmoji($reactionEmoji: String!) {
			userSetReactionEmoji(reactionEmoji: $reactionEmoji)
		}`)
}
