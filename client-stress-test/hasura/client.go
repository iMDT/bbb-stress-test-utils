package hasura

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bbb-stress-test/common"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func init() {
	ReadSubscriptions()
}

func StartUser(user *common.User) {
	// user.Logger.Info("Initializing user!")
	defer user.Logger.Info("User is leaving!")

	user.CreatedTime = time.Now()

	if user.Benchmarking {
		user.BenchmarkingMetrics["name"] = user.Name
		user.BenchmarkingMetrics["created_time"] = user.CreatedTime

		// user.BenchmarkingLogger.Info("Initializing user! Total of users:", common.GetNumOfUsers())
		// user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Initializing user!")
		// defer user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("User is leaving!")
		defer func() {
			// user.Logger.Info("It will write the csv!")
			user.BenchmarkingMetrics["left"] = time.Since(user.CreatedTime)
			user.BenchmarkingMetrics["left_time"] = time.Now()

			// if dataAsJson, err := json.Marshal(user.BenchmarkingMetrics); err == nil {
			// dataAsJson = append(dataAsJson, '\n')

			// file, _ := os.Create("benchmarking.csv")
			// writer := csv.NewWriter(file)
			// defer writer.Flush()

			common.AddBenckmarkingUser(user)
			// common.WriteToCsv(user)

			//if _, err := user.BenchmarkingJsonFile.Write(dataAsJson); err != nil {
			//	user.Logger.Fatal(err)
			//}
			//}
		}()
	}

	// https: // bbb30.bbb.imdt.dev/bigbluebutton/api

	// https: // bbb30.bbb.imdt.dev/api/rest/meetingStaticData

	// Simulate client requests
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	RequestUrlWithCookies(client, user, common.GetApiUrl())
	restUrl := strings.ReplaceAll(common.GetApiUrl(), "/bigbluebutton/api", "/api/rest")
	RequestUrlWithCookies(client, user, restUrl+"/meetingStaticData")
	RequestUrlWithCookies(client, user, restUrl+"/userMetadata")

	// jar, _ := cookiejar.New(nil)
	// newClient := &http.Client{
	// 	Jar: jar,
	// }

	// // header := http.Header{}
	// // header.Add("Cookie", common.GetCookieJSESSIONID(user.ApiCookie))

	// fmt.Println("Meu cookie eh :" + common.GetCookieJSESSIONID(user.ApiCookie))
	// cookieHeader := strings.Split(common.GetCookieJSESSIONID(user.ApiCookie), ";")[0]
	// fmt.Println("Meu cookie eh :" + cookieHeader)

	// // cria a request e injeta o header
	// // req, err := http.NewRequest("GET", "http://127.0.0.1:8090/bigbluebutton/connection/checkGraphqlAuthorization", nil)
	// req, err := http.NewRequest("GET", "https://bbb30.bbb.imdt.dev/api/rest/meetingStaticData", nil)
	// if err != nil {
	// 	panic(err)
	// }
	// // req.Header = header
	// req.Header.Add("Cookie", cookieHeader)
	// req.Header.Add("x-session-token", cookieHeader)

	// resp, err := newClient.Do(req)
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()

	// // ler body se quiser
	// body, _ := io.ReadAll(resp.Body)
	// fmt.Println(string(body))

	// token=t.mqiGKXJAZjw4MxWEX2EZ; language=en; prefs={%22showLineNumbers%22:false%2C%22noColors%22:true}; _ga=GA1.2.2064765318.1754680338; _gid=GA1.2.970251287.1758545151; sessionID=s.8d34651cbb4e0cd41dfe2fdafdff8dc8; _ga_KXLWLVELHN=GS2.2.s1758977977$o101$g0$t1758977977$j60$l0$h0; JSESSIONID=3CD7C0A3D0A06A6ED61782301289431C

	// https: // bbb30.bbb.imdt.dev/api/rest/userMetadata
	// token=t.mqiGKXJAZjw4MxWEX2EZ; language=en; prefs={%22showLineNumbers%22:false%2C%22noColors%22:true}; _ga=GA1.2.2064765318.1754680338; _gid=GA1.2.970251287.1758545151; sessionID=s.8d34651cbb4e0cd41dfe2fdafdff8dc8; _ga_KXLWLVELHN=GS2.2.s1758977977$o101$g0$t1758977977$j60$l0$h0; JSESSIONID=3CD7C0A3D0A06A6ED61782301289431C

	for !EstablishWsConnection(user) {
		time.Sleep(5 * time.Second)
		user.Logger.Info("Trying to connect again.")
	}

	if !user.Benchmarking {
		common.AddConnectedUser()
	}

	if user.Benchmarking {
		user.BenchmarkingMetrics["connection_established"] = time.Since(user.CreatedTime)
		// user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Connection established:")
	}

	defer func() {
		user.WsConnectionClosed = true
		user.WsConnection.Close()
	}()

	go func() {
		for {
			time.Sleep(20 * time.Second)
			if user.WsConnection == nil {
				user.Logger.Info("Waiting to connect")
			} else if !user.ConnAckReceived {
				user.Logger.Info("Waiting Ack")
			} else if !user.Joined {
				user.Logger.Info("Waiting Join")
			} else {
				return
			}

			if user.WsConnection != nil {
				user.Logger.Info("It will try again..")
				user.Problem = true
				user.ConnAckReceived = false
				user.UserJoinMutationId = 0
				EstablishWsConnection(user)
				user.Logger.Info("Connection established!")
				go handleWsMessages(user)
				SendConnectionInitMessage(user)
				user.Logger.Info("Init message sent")
			}
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			SendUpdateConnectionAliveAt(user, GetCurrMessageId(user))
		}
	}()

	if user.Benchmarking && user.Name == "Benchmarking 01" {
		go func() {
			for {
				time.Sleep(1 * time.Second)
				SendPeriodicChatMessage(user, GetCurrMessageId(user))
			}
		}()
	}

	go handleWsMessages(user)
	SendConnectionInitMessage(user)

	if user.Benchmarking {
		if user.Benchmarking && user.Name == "Benchmarking 01" {
			// wait because he is sending periodic msgs
			time.Sleep(999 * time.Second)
		}

		for {
			if user.Joined && user.Pong && user.Chat {
				break
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		time.Sleep(time.Duration(user.TimeToLive) * time.Second)
	}
}

func handleWsMessages(user *common.User) {
	receivedSubscriptionIds := make(map[string]bool)
	done := make(chan struct{})

	defer close(done)
	for {
		_, message, err := user.WsConnection.ReadMessage()
		if err != nil {
			user.Logger.Debugln("read:", err)
			user.Logger.Debugf("%v", message)
			return
		}

		type HasuraMessageInfo struct {
			Type string `json:"type"`
			Id   string `json:"id"`
		}

		type HasuraMessage struct {
			Type    string `json:"type"`
			Id      string `json:"id"`
			Payload struct {
				Data map[string]json.RawMessage `json:"data"`
			} `json:"payload"`
		}

		//{"id":"41",
		//"payload":{"data":{"meeting":[{"__typename":"meeting","breakoutPolicies":{"__typename":"meeting_breakoutPolicies","breakoutRooms":[],"captureNotes":false,"captureNotesFilename":"%%CONFNAME%%","captureSlides":false,"captureSlidesFilename":"%%CONFNAME%%","freeJoin":false,"parentId":"bbb-none","privateChatEnabled":true,"record":false,"sequence":0},"componentsFlags":{"__typename":"meeting_componentFlags","hasBreakoutRoom":false,"hasCaption":false,"hasExternalVideo":false,"hasPoll":false,"hasScreenshare":false,"hasTimer":false},"createdTime":1708950207625,"disabledFeatures":[],"durationInSeconds":0,"extId":"random-6403352","externalVideo":null,"html5InstanceId":null,"isBreakout":false,"lockSettings":{"__typename":"meeting_lockSettings","disableCam":false,"disableMic":false,"disableNotes":false,"disablePrivateChat":false,"disablePublicChat":false,"hasActiveLockSetting":false,"hideUserList":false,"hideViewersAnnotation":false,"hideViewersCursor":false,"webcamsOnlyForModerator":false},"maxPinnedCameras":3,"meetingCameraCap":0,"meetingId":"10bbce770f3adc35c305f0d7cc34cfc115530b5a-1708950207625","name":"random-6403352","notifyRecordingIsOn":false,"presentationUploadExternalDescription":"","presentationUploadExternalUrl":"","recordingPolicies":{"__typename":"meeting_recordingPolicies","allowStartStopRecording":true,"autoStartRecording":false,"keepEvents":false,"record":false},"screenshare":null,"usersPolicies":{"__typename":"v_meeting_usersPolicies","allowModsToEjectCameras":false,"allowModsToUnmuteUsers":false,"authenticatedGuest":true,"guestLobbyMessage":null,"guestPolicy":"ALWAYS_ACCEPT","maxUserConcurrentAccesses":3,"maxUsers":0,"meetingLayout":"CUSTOM_LAYOUT","moderatorsCanMuteAudio":true,"moderatorsCanUnmuteAudio":false,"userCameraCap":3,"webcamsOnlyForModerator":false},"voiceSettings":{"__typename":"meeting_voiceSettings","dialNumber":"613-555-1234","muteOnStart":false,"telVoice":"73939","voiceConf":"73939"}}]}},
		//"type":"data"}

		var msg HasuraMessageInfo
		err = json.Unmarshal(message, &msg)
		if err != nil {
			user.Logger.Println("error on unmarshal message:", err)
			user.Logger.Debugf("%v", msg)
			continue
		}

		user.Logger.Debugf("Received: %s %v", msg.Id, msg)

		switch msg.Type {
		case "connection_ack":
			if user.Problem {
				user.Logger.Infof("Received connection_ack: %s", msg)
			} else {
				user.Logger.Debugln("Received connection_ack")
			}
			if !user.ConnAckReceived {
				if user.Benchmarking {
					user.Logger.Info("Received connection_ack")
					user.BenchmarkingMetrics["connection_ack_received"] = time.Since(user.CreatedTime)
					// user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Received connection_ack:")
				}

				user.ConnAckReceived = true
				SendUserCurrentSubscription(user)
			}
		case "next":
			if user.Problem {
				user.Logger.Infof("Received data Id: %s", msg.Id)
				user.Logger.Infof("Received data: %s", msg)
			} else {
				user.Logger.Debugf("Received data: %s", msg.Id)
			}

			if _, exists := receivedSubscriptionIds[msg.Id]; !exists {
				receivedSubscriptionIds[msg.Id] = true
				common.AddSubscriptionReceived()
			}

			if user.UserJoinMutationId == 0 {
				if user.Benchmarking {
					user.BenchmarkingMetrics["join_start"] = time.Since(user.CreatedTime)
					// user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Will send Join Message:")
				}
				SendJoinMessage(user)
				if user.Benchmarking {
					user.BenchmarkingMetrics["join_sent"] = time.Since(user.CreatedTime)
					// user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Sent Join Message:")
				}
			}

			if msg.Id == fmt.Sprintf("%d", user.ConnectionAliveMutationId) {
				if user.Benchmarking {
					user.Logger.Infoln("Ping successfully DATA.")
					user.BenchmarkingMetrics["connection_alive_completed"] = time.Since(user.CreatedTime)

					user.ChatMessageMutationId = GetCurrMessageId(user)
					go SendSendGroupChatMessageMsg(user, 0, user.ChatMessageMutationId, "I'm here "+user.Name)
				}
				user.Pong = true
				// user.Logger.Infoln("Ping successfully.")

			}

			if msg.Id == fmt.Sprintf("%d", user.UserCurrentSubscriptionId) {
				if user.Benchmarking {
					user.BenchmarkingMetrics["user_current_data_received"] = time.Since(user.CreatedTime)
				}

				user.Logger.Println("Received user_current data")

				var hasuraMessage HasuraMessage
				err = json.Unmarshal(message, &hasuraMessage)
				if err != nil {
					user.Logger.Println("error on unmarshal message:", err)
					user.Logger.Debugf("%v", hasuraMessage)
					continue
				}

				var messagePayloadData []interface{}
				if err := json.Unmarshal(hasuraMessage.Payload.Data["user_current"], &messagePayloadData); err != nil {
					panic(err)
				}

				if len(messagePayloadData) > 0 {
					firstItemOfMessage := messagePayloadData[0]
					if firstItemOfMessageAsMap, currDataOk := firstItemOfMessage.(map[string]interface{}); currDataOk {
						payloadAsJsonByte, _ := json.Marshal(firstItemOfMessageAsMap)
						user.Logger.Debug(string(payloadAsJsonByte))
						if joinedValue, okJoinedValue := firstItemOfMessageAsMap["joined"].(bool); okJoinedValue {
							// user.Logger.Infof("Joined: %t", joinedValue)

							if joinedValue && !user.Joined {
								if user.Benchmarking {
									user.BenchmarkingMetrics["join_received"] = time.Since(user.CreatedTime)
									// user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Joined:")
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

								// Wait for re-connection
								time.Sleep(2 * time.Second)

								//for i := 0; i < 25; i++ {
								//	//time.Sleep(1000 * time.Millisecond)
								//	SendSubscriptionsBatch(user)
								//}

								SendSubscriptionsBatch(user)

								if !user.Benchmarking {
									// SendSubscriptionsBatch(user)
									SendChatMessages(user)
								}
							}
						}
					}
				}
			}

			//payloadAsJsonByte, err := json.Marshal(msg.Payload)
			//if err != nil {
			//	user.Logger.Printf("error marshalling connection_init message: %v", err)
			//	return
			//}

			// fmt.Println(string(payloadAsJsonByte))
			if bytes.Contains(message, []byte("chat_message_public")) {
				SendUpdateChatLastSeenAt(user)

				// check if it is the message I sent
				if bytes.Contains(message, []byte("MY MSG "+strconv.Itoa(user.PeriodicChatMessageCounter))) {
					if bytes.Contains(message, []byte(user.UserId)) {
						user.PeriodicChatMessageRtts = append(user.PeriodicChatMessageRtts, time.Since(user.PeriodicChatMessageSentAt).Milliseconds())
						log.Infof("Received MY MSG, it took: %v milliseconds.\n", time.Since(user.PeriodicChatMessageSentAt).Milliseconds())
						// AVERAGE
						var sumOfRtts int64
						for i := range user.PeriodicChatMessageRtts {
							sumOfRtts += user.PeriodicChatMessageRtts[i]
						}
						log.Infof("Current chat message rtt average: %v milliseconds.\nPeriodic rtts:%v\n", float64(sumOfRtts)/float64(len(user.PeriodicChatMessageRtts)), user.PeriodicChatMessageRtts)

						// user.Logger.Info("Received chat message")
						// user.Logger.Info(string(message))
						user.PeriodicChatMessageMutationId = 0

					}
				}
			}

			// if user.UserCurrentSubscriptionId
		case "ka":
			// nothing
		case "complete":

			if msg.Id == fmt.Sprintf("%d", user.ChatMessageMutationId) {
				if user.Benchmarking {
					user.BenchmarkingMetrics["chat_message_completed"] = time.Since(user.CreatedTime)
					user.Logger.Infoln("Chat successfully COMPLETE.")
					user.Chat = true
				}
			}

			if msg.Id == fmt.Sprintf("%d", user.PeriodicChatMessageMutationId) {
				user.Logger.Infoln("Chat Msg successfully COMPLETE.")
			}

			if msg.Id == fmt.Sprintf("%d", user.ConnectionAliveMutationId) {
				if user.Benchmarking {
					//	user.BenchmarkingMetrics["connection_alive_completed"] = time.Since(user.CreatedTime)
					user.Logger.Infoln("Ping successfully COMPLETE.")
				}
				// user.Pong = true
			}

			if user.Benchmarking {
				if msg.Id == fmt.Sprintf("%d", user.UserJoinMutationId) {
					user.Logger.Info("Join completed")
					user.BenchmarkingMetrics["join_completed"] = time.Since(user.CreatedTime)
				}
			}

			if user.Problem {
				user.Logger.Infof("Completed Id: %s", msg.Id)
				user.Logger.Infof("Completed: %s", msg)
			} else {
				user.Logger.Debugf("Completed: %s", msg.Id)
			}
		case "error":
			user.Logger.Errorf("recv error: %s", message)
		case "ping":
			// send pong
			SendPong(user)
		default:
			user.Logger.Debugf("Received unknown type: %s %s", msg.Id, msg.Type)
		}

	}
}

func EstablishWsConnection(user *common.User) bool {
	user.WsConnectionMutex.Lock()
	defer user.WsConnectionMutex.Unlock()

	if user.WsConnection != nil {
		user.WsConnectionClosed = true
		user.WsConnection.Close()
	}

	header := http.Header{}
	header.Add("Cookie", common.GetCookieJSESSIONID(user.ApiCookie))
	wsDialer := websocket.Dialer{}
	wsConn, _, err := wsDialer.Dial(common.GetHasuraWs(), header)
	if err != nil {
		user.Logger.Error("Error while connection WebSocket:", err)
		return false
	}
	user.WsConnection = wsConn
	user.WsConnectionClosed = false

	return true
}

func SendChatMessages(user *common.User) {
	var currNumOfMsgs int64 = 1

	if !common.GetConfig().SendChatMessages {
		return
	}

	config := common.GetConfig()

	user.Logger.Debugln("Sending chat messages")

	if len(config.ListOfMessages) > 0 {
		numOfMessages := rand.Intn(len(config.ListOfMessages)) + 1
		for i := 0; i < numOfMessages; i++ {
			go SendSendGroupChatMessageMsg(user, GetCurrMessageId(user), GetCurrMessageId(user), strconv.FormatInt(currNumOfMsgs, 10)+" "+config.ListOfMessages[i])
			currNumOfMsgs++
			time.Sleep(time.Duration(config.IntervalBetweenMessagesInMs) * time.Millisecond)
		}
	}
}

func SendConnectionInitMessage(user *common.User) {
	msgConnectionInit := map[string]interface{}{
		"type": "connection_init",
		"payload": map[string]map[string]string{
			"headers": {
				"X-Session-Token":            user.SessionToken,
				"X-ClientSessionUUID":        "myUid" + user.SessionToken,
				"X-ClientType":               "HTML5",
				"X-ClientIsMobile":           "false",
				"X-Session-Benchmarking":     fmt.Sprintf("%t", user.Benchmarking),
				"X-Session-BenchmarkingName": fmt.Sprintf("%s", user.Name),
			},
		},
	}

	WriteToWebSocket(user, msgConnectionInit)
}

func WriteToWebSocket(user *common.User, msg map[string]interface{}) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		user.Logger.Printf("error marshalling connection_init message: %v", err)
		return
	}

	user.WsConnectionMutex.Lock()
	err = user.WsConnection.WriteMessage(websocket.TextMessage, msgBytes)
	user.WsConnectionMutex.Unlock()

	if err != nil {
		user.Logger.Println("write 1:", err)
		return
	}
}

func SendGenericGraphqlMessage(user *common.User, messageId int, variables map[string]interface{}, operationName string, query string) {
	user.Logger.Debugln("Sending " + operationName + " Id: " + strconv.Itoa(messageId))

	subs := struct {
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

	msgBytes, err := json.Marshal(subs)
	if err != nil {
		user.Logger.Errorf("error marshalling message: %v", err)
		return
	}

	user.WsConnectionMutex.Lock()
	err = user.WsConnection.WriteMessage(websocket.TextMessage, msgBytes)
	user.WsConnectionMutex.Unlock()
	if err != nil {
		user.Logger.Errorln("write 2:", err)
		user.Logger.Errorln("message was:", string(msgBytes))
		return
	}
}

func SendSendGroupChatMessageMsg(user *common.User, typingMessageId int, messageId int, chatMessage string) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping groupChatMessage %d because the connection is closed.", user.ConnectionAliveMutationId)
		return
	}

	// Send Typing

	if user.Benchmarking {
		user.Logger.Infoln("Sending chat message " + chatMessage)
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

	// Send Message
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

	//
	//
	//{"id":"6a523c11-5e07-479c-96fd-eb9916f03b7d",
	//	"type":"subscribe",
	//	"payload":{
	//	"variables":{"chatMessageInMarkdownFormat":"ae","chatId":"MAIN-PUBLIC-GROUP-CHAT"},
	//	"extensions":{},"operationName":"ChatSendMessage",
	//		"query":"mutation ChatSendMessage($chatId: String!, $chatMessageInMarkdownFormat: String!) " +
	//		"{\n  chatSendMessage(\n    chatId: $chatId\n    chatMessageInMarkdownFormat: $chatMessageInMarkdownFormat\n  )\n}"}}
}

func SendUpdateConnectionAliveAtBenchmarking(user *common.User) {
	user.ConnectionAliveMutationId = GetCurrMessageId(user)

	user.Logger.Debugf("Created alive at %d", user.ConnectionAliveMutationId)

	SendUpdateConnectionAliveAt(user, user.ConnectionAliveMutationId)
}

func SendPong(user *common.User) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping pong %d because the connection is closed.", user.ConnectionAliveMutationId)
		return
	}

	pongMessage := struct {
		Type    string `json:"type"`
		Payload struct {
			Message string `json:"message"`
		} `json:"payload"`
	}{
		Type: "pong",
		Payload: struct {
			Message string `json:"message"`
		}{
			Message: "keepalive",
		},
	}

	msgBytes, err := json.Marshal(pongMessage)
	if err != nil {
		user.Logger.Errorf("error marshalling message: %v", err)
		return
	}

	user.WsConnectionMutex.Lock()
	err = user.WsConnection.WriteMessage(websocket.TextMessage, msgBytes)
	user.WsConnectionMutex.Unlock()
	if err != nil {
		user.Logger.Errorln("write 2:", err)
		user.Logger.Errorln("message was:", string(msgBytes))
		return
	}
	user.Logger.Debugf("Sent pong.")
}

func SendUpdateConnectionAliveAt(user *common.User, messageId int) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping alive %d because the connection is closed.", user.ConnectionAliveMutationId)
		return
	}

	//{"id":"f84a9a3f-2315-418e-a5b3-a13bf5e689f0","payload":{"data":{"userSetConnectionAlive":true}},"type":"next"}
	//{"id":"cdc84bf1-ef6b-492d-9325-7a61a2b0eca3",
	//"type":"subscribe","payload":
	//{"variables":{"networkRttInMs":9},
	//"extensions":{},"operationName":"UpdateConnectionAliveAt","query":
	//"mutation UpdateConnectionAliveAt($networkRttInMs: Float!) {\n
	//userSetConnectionAlive(networkRttInMs: $networkRttInMs)\n}"}}

	// Send Message
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
		}
	`)
}

func SendPeriodicChatMessage(user *common.User, messageId int) {
	// wait it returned to send the next
	if user.PeriodicChatMessageMutationId == 0 {
		user.PeriodicChatMessageMutationId = GetCurrMessageId(user)
		user.PeriodicChatMessageSentAt = time.Now()
		user.PeriodicChatMessageCounter++
		msgText := "MY MSG " + strconv.Itoa(user.PeriodicChatMessageCounter)
		go SendSendGroupChatMessageMsg(user, 0, user.PeriodicChatMessageMutationId, msgText)
		user.Logger.Infoln("Sent " + msgText)
	}
}

func SendUpdateChatLastSeenAt(user *common.User) {
	if user.WsConnectionClosed {
		user.Logger.Debugf("Skipping chatLasSeen %d because the connection is closed.", user.ConnectionAliveMutationId)
		return
	}

	now := time.Now()

	SendGenericGraphqlMessage(
		user,
		GetCurrMessageId(user),
		map[string]interface{}{
			"chatId":     "MAIN-PUBLIC-GROUP-CHAT",
			"lastSeenAt": now.Format("2006-01-02T15:04:05.999Z"),
		},
		"UpdateChatUser",
		`mutation UpdateChatLastSeen($chatId: String, $lastSeenAt: String) {
				chatSetLastSeen(
				  chatId: $chatId
				  lastSeenAt: $lastSeenAt
				)
			  }
	`)
}

//{"id":"c0e04c63-8d4f-49d0-b596-76b8d4dfb34c",
//"type":"subscribe",
//"operationName":"UpdateChatUser",
//"query":"mutation UpdateChatUser($chatId: String, $lastSeenAt: timestamptz) {
//	\n  update_chat_user(where: {chatId: {_eq: $chatId}, _or: [{lastSeenAt: {_lt: $lastSeenAt}}, {lastSeenAt: {_is_null: true}}]}\n
//	_set: {lastSeenAt: $lastSeenAt}\n  )
//	{\n    affected_rows\n    __typename\n  }\n}"}}

//{"id":"186","type":"start","payload":{"variables":{"networkRttInMs":5.300000000745058},"extensions":{},"operationName":"UpdateConnectionAliveAt","query":"mutation UpdateConnectionAliveAt($networkRttInMs: Float!) {\n  userSetConnectionAlive(networkRttInMs: $networkRttInMs)\n}"}}

func SendUserCurrentSubscription(user *common.User) {
	user.UserCurrentSubscriptionId = GetCurrMessageId(user)

	// 1aa40053-6219-4bd8-8e6d-2d092b139bed
	patternQueryId := `"id":"[\d\w\-]+"`
	reQueryId, errPattern := regexp.Compile(patternQueryId)
	if errPattern != nil {
		fmt.Println("Error compiling regex:", errPattern)
	}

	replacementQueryId := fmt.Sprintf(`"id":"%d"`, user.UserCurrentSubscriptionId)
	userCurrentQueryWithId := reQueryId.ReplaceAllString(userCurrentSubscription, replacementQueryId)

	user.Logger.Debugf("Sending %s", strings.ReplaceAll(userCurrentQueryWithId, "\n", " ")[0:60])
	// user.Logger.Debugf("Sending %s", v)
	// user.Logger.Infoln(v)
	user.WsConnectionMutex.Lock()
	err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(userCurrentQueryWithId))
	user.WsConnectionMutex.Unlock()
	if err != nil {
		user.Logger.Println("write 3:", err)
		return
	}

	common.AddSubscriptionSent()
}

func GetCurrMessageId(user *common.User) int {
	user.CurrMessageId++
	return user.CurrMessageId - 1
}

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
						userJoinMeeting(authToken: $authToken, clientType: $clientType, clientIsMobile: $clientIsMobile,)
				}`)
}

var (
	subscriptions           []string
	userCurrentSubscription string
)

func ReadSubscriptions() {
	dir := "./subscriptions"

	numberOfSubscriptions := 0
	// Walk the directory
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			if fileContent, lastContentErr := ioutil.ReadFile(path); lastContentErr == nil && string(fileContent) != "" {

				numberOfSubscriptions++
				if strings.Contains(string(fileContent), "Patched_userCurrentSubscription") {
					userCurrentSubscription = string(fileContent)
					return nil
				}

				subscriptions = append(subscriptions, string(fileContent))
			}
		}

		return nil
	})
	if err != nil {
		fmt.Println("Error walking through directory:", err)
	}

	fmt.Printf("%d subscriptions found.\n", numberOfSubscriptions)
}

func SendSubscriptionsBatch(user *common.User) {
	if !common.GetConfig().SendSubscriptionsBatch {
		return
	}

	user.Logger.Debugln("Sending Hasura subscriptions batch")

	// 1aa40053-6219-4bd8-8e6d-2d092b139bed
	patternQueryId := `"id":"[\d\w\-]+"`
	reQueryId, errPattern := regexp.Compile(patternQueryId)
	if errPattern != nil {
		fmt.Println("Error compiling regex:", errPattern)
	}

	patternUserId := `"userId":"\d+"`
	reUserId, errPattern := regexp.Compile(patternUserId)
	if errPattern != nil {
		fmt.Println("Error compiling regex:", errPattern)
	}

	for _, v := range subscriptions {

		replacementQueryId := fmt.Sprintf(`"id":"%d"`, GetCurrMessageId(user))
		textFromFileWithNewId := reQueryId.ReplaceAllString(v, replacementQueryId)

		replacementUserId := fmt.Sprintf(`"userId":"%s"`, user.UserId)
		textFromFileWithNewId = reUserId.ReplaceAllString(textFromFileWithNewId, replacementUserId)

		user.Logger.Debugf("Sending %s", strings.ReplaceAll(textFromFileWithNewId, "\n", " ")[0:60])
		// user.Logger.Debugf("Sending %s", v)
		// user.Logger.Infoln(v)
		user.WsConnectionMutex.Lock()
		err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(textFromFileWithNewId))
		user.WsConnectionMutex.Unlock()
		if err != nil {
			user.Logger.Println("write 3:", err)
			return
		}
		common.AddSubscriptionSent()
		// time.Sleep(3 * time.Second)
	}
}

func RequestUrlWithCookies(client *http.Client, user *common.User, url string) {
	user.Logger.Debugln(url)
	// Create a new HTTP request to the authentication hook URL.
	req, _ := http.NewRequest("GET", url, nil)
	// Add cookies to the request.
	for _, cookie := range user.ApiCookie {
		req.AddCookie(cookie)
	}

	// Execute the HTTP request to obtain user session variables (like X-Hasura-Role)
	// req.Header.Set("x-original-uri", authHookUrl+"?sessionToken="+sessionToken)
	req.Header.Set("x-session-token", user.SessionToken)
	// req.Header.Set("User-Agent", "hasura-graphql-engine")
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	user.Logger.Traceln(string(respBody))
}
