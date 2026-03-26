package hasura

import (
	"bbb-stress-test/common"
	"math/rand"
	"strconv"
	"time"
)

// SendJoinMessage sends the userJoinMeeting GraphQL mutation.
func SendJoinMessage(user *common.User) {
	user.UserJoinMutationId = GetCurrMessageId(user)
	SendGenericGraphqlMessage(
		user,
		user.UserJoinMutationId,
		map[string]interface{}{
			"authToken":      user.AuthToken,
			"clientType":     "HTML5",
			"clientIsMobile": false,
		},
		"UserJoin",
		`mutation UserJoin($authToken: String!, $clientType: String!, $clientIsMobile: Boolean!) {
			userJoinMeeting(authToken: $authToken, clientType: $clientType, clientIsMobile: $clientIsMobile)
		}`)
}

// SendUpdateConnectionAliveAt sends a userSetConnectionAlive mutation to keep
// the server session alive.
func SendUpdateConnectionAliveAt(user *common.User, messageId int) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping alive %d: connection is closed.", messageId)
		return
	}
	SendGenericGraphqlMessage(
		user,
		messageId,
		map[string]interface{}{
			"serverRequestId":    "abcd",
			"clientSessionUUID":  user.UserId + "-clientuid",
			"networkRttInMs":     5,
			"applicationRttInMs": 0,
			"traceLog":           "",
		},
		"UpdateConnectionAliveAt",
		`mutation UpdateConnectionAliveAt($serverRequestId: String!, $clientSessionUUID: String!, $networkRttInMs: Float!, $applicationRttInMs: Float, $traceLog: String) {
			userSetConnectionAlive(
			  serverRequestId: $serverRequestId
			  clientSessionUUID: $clientSessionUUID
			  networkRttInMs: $networkRttInMs
			  applicationRttInMs: $applicationRttInMs
			  traceLog: $traceLog
			)
		}`)
}

// SendUpdateConnectionAliveAtBenchmarking allocates a new message ID, records
// it on the user, and sends a connection-alive mutation.
func SendUpdateConnectionAliveAtBenchmarking(user *common.User) {
	user.ConnectionAliveMutationId = GetCurrMessageId(user)
	SendUpdateConnectionAliveAt(user, user.ConnectionAliveMutationId)
}

// SendUpdateChatLastSeenAt marks the main public chat as seen up to now.
func SendUpdateChatLastSeenAt(user *common.User) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping chatLastSeen: connection is closed.")
		return
	}
	SendGenericGraphqlMessage(
		user,
		GetCurrMessageId(user),
		map[string]interface{}{
			"chatId":     "MAIN-PUBLIC-GROUP-CHAT",
			"lastSeenAt": time.Now().Format("2006-01-02T15:04:05.999Z"),
		},
		"UpdateChatUser",
		`mutation UpdateChatLastSeen($chatId: String, $lastSeenAt: String) {
			chatSetLastSeen(
			  chatId: $chatId
			  lastSeenAt: $lastSeenAt
			)
		}`)
}

// SendSendGroupChatMessageMsg sends a chat message to the main public group
// chat. If typingMessageId is non-zero, a ChatSetTyping mutation is sent first.
func SendSendGroupChatMessageMsg(user *common.User, typingMessageId, messageId int, chatMessage string) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping groupChatMessage: connection is closed.")
		return
	}

	if user.Benchmarking {
		user.Logger.Infoln("Sending chat message: " + chatMessage)
	}

	if typingMessageId != 0 {
		SendGenericGraphqlMessage(
			user,
			typingMessageId,
			map[string]interface{}{
				"chatId": "MAIN-PUBLIC-GROUP-CHAT",
			},
			"ChatSetTyping",
			`mutation ChatSetTyping($chatId: String!) { chatSetTyping(chatId: $chatId) }`)
		time.Sleep(1 * time.Second)
	}

	SendGenericGraphqlMessage(
		user,
		messageId,
		map[string]interface{}{
			"chatMessageInMarkdownFormat": chatMessage,
			"chatId":                      "MAIN-PUBLIC-GROUP-CHAT",
		},
		"ChatSendMessage",
		`mutation ChatSendMessage($chatId: String!, $chatMessageInMarkdownFormat: String!) {
			chatSendMessage(
				chatId: $chatId
				chatMessageInMarkdownFormat: $chatMessageInMarkdownFormat
			)
		}`)
}

// SendChatMessages sends a random subset of the configured messages from the
// user. It is a no-op when SendChatMessages is disabled in the config.
func SendChatMessages(user *common.User) {
	config := common.GetConfig()
	if !config.SendChatMessages || len(config.ListOfMessages) == 0 {
		return
	}

	user.Logger.Debugln("Sending chat messages")

	var msgCounter int64 = 1
	numOfMessages := rand.Intn(len(config.ListOfMessages)) + 1
	for i := 0; i < numOfMessages; i++ {
		go SendSendGroupChatMessageMsg(
			user,
			GetCurrMessageId(user),
			GetCurrMessageId(user),
			strconv.FormatInt(msgCounter, 10)+" "+config.ListOfMessages[i],
		)
		msgCounter++
		time.Sleep(time.Duration(config.IntervalBetweenMessagesInMs) * time.Millisecond)
	}
}

// SendPeriodicChatMessage sends the next benchmarking chat message if the
// previous one has already completed (PeriodicChatMessageMutationId == 0).
func SendPeriodicChatMessage(user *common.User, messageId int) {
	if user.PeriodicChatMessageMutationId != 0 {
		return
	}
	user.PeriodicChatMessageMutationId = GetCurrMessageId(user)
	user.PeriodicChatMessageSentAt = time.Now()
	user.PeriodicChatMessageCounter++
	msgText := "MY MSG " + strconv.Itoa(user.PeriodicChatMessageCounter)
	go SendSendGroupChatMessageMsg(user, 0, user.PeriodicChatMessageMutationId, msgText)
	user.Logger.Infoln("Sent " + msgText)
}
