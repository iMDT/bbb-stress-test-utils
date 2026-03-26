package hasura

import (
	"bbb-stress-test/common"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

// EstablishWsConnection dials the GraphQL WebSocket endpoint and stores the
// connection on the user. Returns false if the dial fails.
func EstablishWsConnection(user *common.User) bool {
	user.WsConnectionMutex.Lock()
	defer user.WsConnectionMutex.Unlock()

	if user.WsConnection != nil {
		user.WsConnectionClosed = true
		user.WsConnection.Close()
	}

	header := http.Header{}
	header.Add("Cookie", common.GetCookieJSESSIONID(user.ApiCookie))

	wsDialer := &websocket.Dialer{}
	wsConn, _, err := wsDialer.Dial(common.GetHasuraWs(), header)
	if err != nil {
		user.Logger.Error("Error connecting WebSocket:", err)
		return false
	}

	user.WsConnection = wsConn
	user.WsConnectionClosed = false
	return true
}

// SendConnectionInitMessage sends the GraphQL-over-WebSocket connection_init
// message, including the session headers required by BBB.
func SendConnectionInitMessage(user *common.User) {
	writeWsMessage(user, map[string]interface{}{
		"type": "connection_init",
		"payload": map[string]map[string]string{
			"headers": {
				"X-Session-Token":            user.SessionToken,
				"X-ClientSessionUUID":        "myUid" + user.SessionToken,
				"X-ClientType":               "HTML5",
				"X-ClientIsMobile":           "false",
				"X-Session-Benchmarking":     fmt.Sprintf("%t", user.Benchmarking),
				"X-Session-BenchmarkingName": user.Name,
			},
		},
	})
}

// SendGenericGraphqlMessage sends a GraphQL subscribe/mutation message with
// the given ID, variables, operation name, and query string.
func SendGenericGraphqlMessage(user *common.User, messageId int, variables map[string]interface{}, operationName, query string) {
	user.Logger.Debugln("Sending " + operationName + " Id: " + strconv.Itoa(messageId))

	msg := struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Payload struct {
			Variables     map[string]interface{} `json:"variables"`
			Extensions    map[string]interface{} `json:"extensions"`
			OperationName string                 `json:"operationName"`
			Query         string                 `json:"query"`
		} `json:"payload"`
	}{
		ID:   fmt.Sprintf("%d", messageId),
		Type: "subscribe",
		Payload: struct {
			Variables     map[string]interface{} `json:"variables"`
			Extensions    map[string]interface{} `json:"extensions"`
			OperationName string                 `json:"operationName"`
			Query         string                 `json:"query"`
		}{
			Variables:     variables,
			Extensions:    make(map[string]interface{}),
			OperationName: operationName,
			Query:         query,
		},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		user.Logger.Errorf("error marshalling message: %v", err)
		return
	}

	writeRawWsMessage(user, msgBytes)
}

// SendPong responds to a server ping with a keepalive pong.
func SendPong(user *common.User) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping pong: connection is closed.")
		return
	}

	pong := struct {
		Type    string `json:"type"`
		Payload struct {
			Message string `json:"message"`
		} `json:"payload"`
	}{
		Type: "pong",
		Payload: struct {
			Message string `json:"message"`
		}{Message: "keepalive"},
	}

	msgBytes, err := json.Marshal(pong)
	if err != nil {
		user.Logger.Errorf("error marshalling pong: %v", err)
		return
	}

	writeRawWsMessage(user, msgBytes)
	user.Logger.Debugf("Sent pong.")
}

// GetCurrMessageId atomically increments and returns the next message ID for
// the user. Message IDs are monotonically increasing integers.
func GetCurrMessageId(user *common.User) int {
	user.CurrMessageId++
	return user.CurrMessageId - 1
}

// writeWsMessage marshals msg and writes it to the WebSocket connection.
func writeWsMessage(user *common.User, msg interface{}) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		user.Logger.Printf("error marshalling message: %v", err)
		return
	}
	writeRawWsMessage(user, msgBytes)
}

// writeRawWsMessage writes pre-serialized bytes to the WebSocket connection,
// holding the user mutex to prevent concurrent writes.
func writeRawWsMessage(user *common.User, msgBytes []byte) {
	user.WsConnectionMutex.Lock()
	err := user.WsConnection.WriteMessage(websocket.TextMessage, msgBytes)
	user.WsConnectionMutex.Unlock()

	if err != nil {
		user.Logger.Println("ws write error:", err)
	}
}
