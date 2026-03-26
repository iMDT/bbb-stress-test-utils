package hasura

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"bbb-stress-test/common"

	log "github.com/sirupsen/logrus"
)

// hasuraMessageInfo holds the minimal fields needed to dispatch incoming messages.
type hasuraMessageInfo struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

// hasuraMessage is the full message shape, used when payload data is needed.
type hasuraMessage struct {
	Type    string `json:"type"`
	Id      string `json:"id"`
	Payload struct {
		Data map[string]json.RawMessage `json:"data"`
	} `json:"payload"`
}

// handleWsMessages reads messages from the WebSocket connection and dispatches
// them by type. It runs until the connection is closed or an error occurs.
func handleWsMessages(user *common.User) {
	receivedSubscriptionIds := make(map[string]bool)

	for {
		_, message, err := user.WsConnection.ReadMessage()
		if err != nil {
			user.Logger.Debugln("read error:", err)
			return
		}

		var msg hasuraMessageInfo
		if err = json.Unmarshal(message, &msg); err != nil {
			user.Logger.Println("error unmarshalling message:", err)
			continue
		}

		user.Logger.Debugf("Received type=%s id=%s", msg.Type, msg.Id)

		switch msg.Type {
		case "connection_ack":
			handleConnectionAck(user, msg)
		case "next":
			handleNext(user, message, msg.Id, receivedSubscriptionIds)
		case "complete":
			handleComplete(user, msg)
		case "ping":
			SendPong(user)
		case "error":
			user.Logger.Errorf("recv error: %s", message)
		case "ka":
			// server keepalive — no action needed
		default:
			user.Logger.Debugf("Unknown message type: %s id=%s", msg.Type, msg.Id)
		}
	}
}

func handleConnectionAck(user *common.User, msg hasuraMessageInfo) {
	if user.Problem {
		user.Logger.Infof("Received connection_ack: %v", msg)
	} else {
		user.Logger.Debugln("Received connection_ack")
	}

	if user.ConnAckReceived {
		return
	}

	if user.Benchmarking {
		user.Logger.Info("Received connection_ack")
		user.BenchmarkingMetrics["connection_ack_received"] = time.Since(user.CreatedTime)
	}

	user.ConnAckReceived = true
	SendUserCurrentSubscription(user)
}

func handleNext(user *common.User, message []byte, msgId string, receivedSubscriptionIds map[string]bool) {
	if user.Problem {
		user.Logger.Infof("Received data id=%s", msgId)
	} else {
		user.Logger.Debugf("Received data id=%s", msgId)
	}

	if _, seen := receivedSubscriptionIds[msgId]; !seen {
		receivedSubscriptionIds[msgId] = true
		common.AddSubscriptionReceived()
	}

	if user.UserJoinMutationId == 0 {
		if user.Benchmarking {
			user.BenchmarkingMetrics["join_start"] = time.Since(user.CreatedTime)
		}
		SendJoinMessage(user)
		if user.Benchmarking {
			user.BenchmarkingMetrics["join_sent"] = time.Since(user.CreatedTime)
		}
	}

	if msgId == strconv.Itoa(user.ConnectionAliveMutationId) {
		if user.Benchmarking {
			user.Logger.Infoln("Ping successfully DATA.")
			user.BenchmarkingMetrics["connection_alive_completed"] = time.Since(user.CreatedTime)
			user.ChatMessageMutationId = GetCurrMessageId(user)
			go SendSendGroupChatMessageMsg(user, 0, user.ChatMessageMutationId, "I'm here "+user.Name)
		}
		user.Pong = true
	}

	if msgId == strconv.Itoa(user.UserCurrentSubscriptionId) {
		handleUserCurrentData(user, message)
	}

	if bytes.Contains(message, []byte("chat_message_public")) {
		handleChatMessagePublic(user, message)
	}

	if id, err := strconv.Atoi(msgId); err == nil {
		if opName, tracked := user.SubscriptionNames[id]; tracked && isMeetingIdValidationTarget(opName) {
			validateSubscriptionMeetingIds(user, message, opName)
		}
	}
}

func handleUserCurrentData(user *common.User, message []byte) {
	if user.Benchmarking {
		user.BenchmarkingMetrics["user_current_data_received"] = time.Since(user.CreatedTime)
	}

	user.Logger.Println("Received user_current data")

	var fullMsg hasuraMessage
	if err := json.Unmarshal(message, &fullMsg); err != nil {
		user.Logger.Println("error unmarshalling user_current message:", err)
		return
	}

	var payloadItems []interface{}
	if err := json.Unmarshal(fullMsg.Payload.Data["user_current"], &payloadItems); err != nil {
		panic(err)
	}

	if len(payloadItems) == 0 {
		return
	}

	firstItem, ok := payloadItems[0].(map[string]interface{})
	if !ok {
		return
	}

	payloadBytes, _ := json.Marshal(firstItem)
	user.Logger.Debug(string(payloadBytes))

	joined, ok := firstItem["joined"].(bool)
	if !ok || !joined || user.Joined {
		return
	}

	if user.Benchmarking {
		user.BenchmarkingMetrics["join_received"] = time.Since(user.CreatedTime)
	}

	user.Logger.Infoln("Joined successfully.")
	user.Joined = true
	user.Problem = false

	if !user.Benchmarking {
		common.AddJoinedUser()
	}

	if user.Benchmarking {
		SendUpdateConnectionAliveAtBenchmarking(user)
	}

	time.Sleep(2 * time.Second)
	SendSubscriptionsBatch(user)

	if !user.Benchmarking {
		SendChatMessages(user)
	}
}

func handleChatMessagePublic(user *common.User, message []byte) {
	SendUpdateChatLastSeenAt(user)

	expectedMsg := "MY MSG " + strconv.Itoa(user.PeriodicChatMessageCounter)
	if !bytes.Contains(message, []byte(expectedMsg)) {
		return
	}
	if !bytes.Contains(message, []byte(user.UserId)) {
		return
	}

	rtt := time.Since(user.PeriodicChatMessageSentAt).Milliseconds()
	user.PeriodicChatMessageRtts = append(user.PeriodicChatMessageRtts, rtt)
	log.Infof("Received MY MSG, RTT: %v ms.\n", rtt)

	var sumRtts int64
	for _, r := range user.PeriodicChatMessageRtts {
		sumRtts += r
	}
	avg := float64(sumRtts) / float64(len(user.PeriodicChatMessageRtts))
	log.Infof("Chat RTT average: %.1f ms. All RTTs: %v\n", avg, user.PeriodicChatMessageRtts)

	user.PeriodicChatMessageMutationId = 0
}

func handleComplete(user *common.User, msg hasuraMessageInfo) {
	if user.Problem {
		user.Logger.Infof("Completed id=%s", msg.Id)
	} else {
		user.Logger.Debugf("Completed id=%s", msg.Id)
	}

	if msg.Id == fmt.Sprintf("%d", user.ChatMessageMutationId) && user.Benchmarking {
		user.BenchmarkingMetrics["chat_message_completed"] = time.Since(user.CreatedTime)
		user.Logger.Infoln("Chat message COMPLETE.")
		user.Chat = true
	}

	if msg.Id == fmt.Sprintf("%d", user.PeriodicChatMessageMutationId) {
		user.Logger.Infoln("Periodic chat message COMPLETE.")
	}

	if msg.Id == fmt.Sprintf("%d", user.ConnectionAliveMutationId) && user.Benchmarking {
		user.Logger.Infoln("Ping COMPLETE.")
	}

	if user.Benchmarking && msg.Id == fmt.Sprintf("%d", user.UserJoinMutationId) {
		user.Logger.Info("Join COMPLETE.")
		user.BenchmarkingMetrics["join_completed"] = time.Since(user.CreatedTime)
	}
}

// meetingIdValidationTargets lists the subscription operation names whose
// incoming data should be validated against the user's expected meetingId.
var meetingIdValidationTargets = map[string]bool{
	"Patched_VideoStreams":               true,
	"Patched_UserListSubscription":       true,
	"whiteboardCursorAccessSubscription": true,
}

func isMeetingIdValidationTarget(operationName string) bool {
	return meetingIdValidationTargets[operationName]
}

// validateSubscriptionMeetingIds inspects every item returned in the "next"
// payload and warns if any item's meetingId does not match the user's meeting.
// Items without a meetingId field are silently skipped.
func validateSubscriptionMeetingIds(user *common.User, message []byte, operationName string) {
	var fullMsg hasuraMessage
	if err := json.Unmarshal(message, &fullMsg); err != nil {
		user.Logger.Warnf("[%s] failed to parse message for meetingId validation: %v", operationName, err)
		return
	}

	for dataKey, rawData := range fullMsg.Payload.Data {
		var items []map[string]interface{}
		if err := json.Unmarshal(rawData, &items); err != nil {
			continue // field is not an array (e.g. aggregate counts)
		}

		for _, item := range items {
			receivedMeetingId, ok := item["meetingId"].(string)
			if !ok {
				continue // item has no meetingId field
			}
			if receivedMeetingId != user.MeetingId {
				user.Logger.Warnf(
					"[%s] meetingId mismatch in %s: got %q, expected %q",
					operationName, dataKey, receivedMeetingId, user.MeetingId,
				)
			}
		}
	}
}
