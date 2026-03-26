package hasura

import (
	"bbb-stress-test/common"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

var (
	subscriptions           []string
	userCurrentSubscription string
)

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
		if strings.Contains(string(content), "Patched_userCurrentSubscription") {
			userCurrentSubscription = string(content)
		} else {
			subscriptions = append(subscriptions, string(content))
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error walking subscriptions directory:", err)
	}
	fmt.Printf("%d subscriptions found.\n", count)
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
		text := reQueryId.ReplaceAllString(v, fmt.Sprintf(`"id":"%d"`, GetCurrMessageId(user)))
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
