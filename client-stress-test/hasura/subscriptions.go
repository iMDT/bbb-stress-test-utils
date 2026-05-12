package hasura

import (
	"bbb-stress-test/common"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

var (
	subscriptions                []string
	pageIdDependentSubscriptions []string
	userCurrentSubscription      string

	rePageIdValue = regexp.MustCompile(`"pageId"\s*:\s*"[^"]*"`)
)

// pageIdParamMarker matches the GraphQL operation declaration when a query
// takes a pageId variable, e.g. `query Foo($pageId: String!)`.
const pageIdParamMarker = "$pageId: String!"

// currentPresentationPagesOpName is the operation name of the subscription
// that publishes the current presentation page; the pageId it returns feeds
// the subscriptions in pageIdDependentSubscriptions.
const currentPresentationPagesOpName = "CurrentPresentationPagesSubscription"

func init() {
	ReadSubscriptions()
}

// ReadSubscriptions loads all GraphQL subscription/query files from the
// ./subscriptions directory. Files containing "Patched_userCurrentSubscription"
// are treated as the user-current subscription; all others are queued for batch
// sending after join.
func ReadSubscriptions() {
	dir := "./subscriptions"
	count := 0

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".txt") {
			return nil
		}

		content, readErr := ioutil.ReadFile(path)
		if readErr != nil || len(content) == 0 {
			return nil
		}

		count++
		s := string(content)
		switch {
		case strings.Contains(s, "Patched_userCurrentSubscription"):
			userCurrentSubscription = s
		case strings.Contains(s, pageIdParamMarker):
			// Defer: needs a pageId from CurrentPresentationPagesSubscription.
			pageIdDependentSubscriptions = append(pageIdDependentSubscriptions, s)
		default:
			subscriptions = append(subscriptions, s)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error walking subscriptions directory:", err)
	}
	fmt.Printf("%d subscriptions found.\n", count)
}

// recordSubscriptionName parses the operationName from the raw subscription
// JSON and stores the msgId → operationName mapping on the user. This lets the
// message handler identify which subscription a "next" message belongs to.
func recordSubscriptionName(user *common.User, msgId int, rawSubscription string) {
	var parsed struct {
		Payload struct {
			OperationName string `json:"operationName"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(rawSubscription), &parsed); err == nil && parsed.Payload.OperationName != "" {
		user.SubscriptionNames[msgId] = parsed.Payload.OperationName
	}
}

// SendUserCurrentSubscription sends the userCurrent subscription that the
// server uses to notify the client when the user has fully joined the meeting.
func SendUserCurrentSubscription(user *common.User) {
	user.UserCurrentSubscriptionId = GetCurrMessageId(user)

	reQueryId := regexp.MustCompile(`"id":"[\d\w\-]+"`)
	replacement := fmt.Sprintf(`"id":"%d"`, user.UserCurrentSubscriptionId)
	query := reQueryId.ReplaceAllString(userCurrentSubscription, replacement)

	user.Logger.Debugf("Sending %s", strings.ReplaceAll(query, "\n", " ")[:60])

	user.WsConnectionMutex.Lock()
	err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(query))
	user.WsConnectionMutex.Unlock()
	if err != nil {
		user.Logger.Println("write error (userCurrentSubscription):", err)
		return
	}

	common.AddSubscriptionSent()
}

// SendSubscriptionsBatch replays all loaded subscription files, substituting
// the user's current ID and a fresh message ID into each one.
func SendSubscriptionsBatch(user *common.User) {
	if !common.GetConfig().SendSubscriptionsBatch {
		return
	}

	user.Logger.Debugln("Sending Hasura subscriptions batch")

	reQueryId := regexp.MustCompile(`"id":"[\d\w\-]+"`)
	reUserId := regexp.MustCompile(`"userId":"\d+"`)

	for _, v := range subscriptions {
		msgId := GetCurrMessageId(user)
		recordSubscriptionName(user, msgId, v)

		if strings.Contains(v, currentPresentationPagesOpName) {
			user.CurrentPresentationPagesSubscriptionId = msgId
		}

		text := reQueryId.ReplaceAllString(v, fmt.Sprintf(`"id":"%d"`, msgId))
		text = reUserId.ReplaceAllString(text, fmt.Sprintf(`"userId":"%s"`, user.UserId))

		user.Logger.Debugf("Sending %s", strings.ReplaceAll(text, "\n", " ")[:60])

		user.WsConnectionMutex.Lock()
		err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(text))
		user.WsConnectionMutex.Unlock()
		if err != nil {
			user.Logger.Println("write error (subscriptions batch):", err)
			return
		}
		common.AddSubscriptionSent()
	}
}

// SendPageIdDependentSubscriptions sends the subscriptions that take a pageId
// variable, substituting the pageId discovered from
// CurrentPresentationPagesSubscription. The caller is responsible for
// cancelling any previously-active deferred subscriptions before invoking
// this with a new pageId (see SendSubscriptionComplete).
func SendPageIdDependentSubscriptions(user *common.User, pageId string) {
	if pageId == "" {
		return
	}
	user.CurrentPageId = pageId
	user.ActivePageIdSubMsgIds = user.ActivePageIdSubMsgIds[:0]

	reQueryId := regexp.MustCompile(`"id":"[\d\w\-]+"`)
	reUserId := regexp.MustCompile(`"userId":"\d+"`)
	pageIdReplacement := fmt.Sprintf(`"pageId":"%s"`, pageId)

	for _, v := range pageIdDependentSubscriptions {
		msgId := GetCurrMessageId(user)
		recordSubscriptionName(user, msgId, v)

		text := reQueryId.ReplaceAllString(v, fmt.Sprintf(`"id":"%d"`, msgId))
		text = reUserId.ReplaceAllString(text, fmt.Sprintf(`"userId":"%s"`, user.UserId))
		text = rePageIdValue.ReplaceAllString(text, pageIdReplacement)

		user.Logger.Debugf("Sending pageId-dependent %s", strings.ReplaceAll(text, "\n", " ")[:60])

		user.WsConnectionMutex.Lock()
		err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(text))
		user.WsConnectionMutex.Unlock()
		if err != nil {
			user.Logger.Println("write error (pageId-dependent subscriptions):", err)
			return
		}
		user.ActivePageIdSubMsgIds = append(user.ActivePageIdSubMsgIds, msgId)
		common.AddSubscriptionSent()
	}
}

// SendSubscriptionComplete sends a graphql-ws "complete" frame to cancel an
// active subscription identified by msgId.
func SendSubscriptionComplete(user *common.User, msgId int) {
	payload := fmt.Sprintf(`{"id":"%d","type":"complete"}`, msgId)

	user.WsConnectionMutex.Lock()
	err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(payload))
	user.WsConnectionMutex.Unlock()
	if err != nil {
		user.Logger.Println("write error (complete):", err)
	}
}
