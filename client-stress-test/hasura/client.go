package hasura

import (
	"bbb-stress-test/common"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func StartUser(user *common.User) {
	//user.Logger.Info("Initializing user!")
	defer user.Logger.Info("User is leaving!")

	user.CreatedTime = time.Now()

	if user.Benchmarking {
		user.BenchmarkingMetrics["name"] = user.Name
		user.BenchmarkingMetrics["created_time"] = user.CreatedTime

		//user.BenchmarkingLogger.Info("Initializing user! Total of users:", common.GetNumOfUsers())
		//user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Initializing user!")
		//defer user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("User is leaving!")
		defer func() {
			//user.Logger.Info("It will write the csv!")
			user.BenchmarkingMetrics["left"] = time.Since(user.CreatedTime)
			user.BenchmarkingMetrics["left_time"] = time.Now()

			//if dataAsJson, err := json.Marshal(user.BenchmarkingMetrics); err == nil {
			//dataAsJson = append(dataAsJson, '\n')

			//file, _ := os.Create("benchmarking.csv")
			//writer := csv.NewWriter(file)
			//defer writer.Flush()

			common.AddBenckmarkingUser(user)
			//common.WriteToCsv(user)

			//if _, err := user.BenchmarkingJsonFile.Write(dataAsJson); err != nil {
			//	user.Logger.Fatal(err)
			//}
			//}
		}()
	}

	for !EstablishWsConnection(user) {
		time.Sleep(5 * time.Second)
		user.Logger.Info("Trying to connect again.")
	}

	if !user.Benchmarking {
		common.AddConnectedUser()
	}

	if user.Benchmarking {
		user.BenchmarkingMetrics["connection_established"] = time.Since(user.CreatedTime)
		//user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Connection established:")
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
			//wait because he is sending periodic msgs
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
					//user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Received connection_ack:")
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
					//user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Will send Join Message:")
				}
				SendJoinMessage(user)
				if user.Benchmarking {
					user.BenchmarkingMetrics["join_sent"] = time.Since(user.CreatedTime)
					//user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Sent Join Message:")
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
				//user.Logger.Infoln("Ping successfully.")

			}

			if msg.Id == fmt.Sprintf("%d", user.UserCurrentSubscriptionId) {
				if user.Benchmarking {
					user.BenchmarkingMetrics["user_current_data_received"] = time.Since(user.CreatedTime)
				}

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

				//currentDataProp := "user_current"
				//payloadDataAsMap := messagePayloadData.([]interface{})

				if len(messagePayloadData) > 0 {
					firstItemOfMessage := messagePayloadData[0]
					if firstItemOfMessageAsMap, currDataOk := firstItemOfMessage.(map[string]interface{}); currDataOk {
						if joinedValue, okJoinedValue := firstItemOfMessageAsMap["joined"].(bool); okJoinedValue {
							//user.Logger.Infof("Joined: %t", joinedValue)

							if joinedValue && !user.Joined {
								if user.Benchmarking {
									user.BenchmarkingMetrics["join_received"] = time.Since(user.CreatedTime)
									//user.BenchmarkingLogger.WithField("1timeSince", fmt.Sprintf("%s", time.Since(user.CreatedTime))).Info("Joined:")
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

								//Wait for re-connection
								time.Sleep(2 * time.Second)

								//for i := 0; i < 25; i++ {
								//	//time.Sleep(1000 * time.Millisecond)
								//	SendSubscriptionsBatch(user)
								//}

								SendSubscriptionsBatch(user)

								if !user.Benchmarking {
									//SendSubscriptionsBatch(user)
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

			//fmt.Println(string(payloadAsJsonByte))
			if bytes.Contains(message, []byte("chat_message_public")) {
				SendUpdateChatLastSeenAt(user)

				//check if it is the message I sent
				if bytes.Contains(message, []byte("MY MSG "+strconv.Itoa(user.PeriodicChatMessageCounter))) {

					if bytes.Contains(message, []byte(user.UserId)) {
						user.PeriodicChatMessageRtts = append(user.PeriodicChatMessageRtts, time.Since(user.PeriodicChatMessageSentAt).Milliseconds())
						log.Infof("Received MY MSG, it took: %v milliseconds.\n", time.Since(user.PeriodicChatMessageSentAt).Milliseconds())
						//AVERAGE
						var sumOfRtts int64
						for i := range user.PeriodicChatMessageRtts {
							sumOfRtts += user.PeriodicChatMessageRtts[i]
						}
						log.Infof("Current chat message rtt average: %v milliseconds.\nPeriodic rtts:%v\n", float64(sumOfRtts)/float64(len(user.PeriodicChatMessageRtts)), user.PeriodicChatMessageRtts)

						//user.Logger.Info("Received chat message")
						//user.Logger.Info(string(message))
						user.PeriodicChatMessageMutationId = 0

					}
				}
			}

			//if user.UserCurrentSubscriptionId
		case "ka":
			//nothing
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
				//user.Pong = true

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

	//Send Typing

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

	//Send Message
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

	//Send Message
	SendGenericGraphqlMessage(
		user,
		messageId,
		map[string]interface{}{
			"networkRttInMs": 5,
		},
		"UpdateConnectionAliveAt",
		`mutation UpdateConnectionAliveAt($networkRttInMs: Float!) { 
					userSetConnectionAlive(networkRttInMs: $networkRttInMs)
				}
	`)
}

func SendPeriodicChatMessage(user *common.User, messageId int) {
	//wait it returned to send the next
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

	SendGenericGraphqlMessage(
		user,
		user.UserCurrentSubscriptionId,
		make(map[string]interface{}),
		"userCurrentSubscriptionStressTest",
		`subscription userCurrentSubscriptionStressTest { user_current { authed banned joined __typename } }`)

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

func SendSubscriptionsBatch(user *common.User) {
	if !common.GetConfig().SendSubscriptionsBatch {
		return
	}

	user.Logger.Debugln("Sending Hasura subscriptions batch")

	//time.Sleep(1 * time.Second)

	var subscriptions []string

	dir := "./subscriptions"

	//1aa40053-6219-4bd8-8e6d-2d092b139bed
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

	// Walk the directory
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			if fileContent, lastContentErr := ioutil.ReadFile(path); lastContentErr == nil && string(fileContent) != "" {
				replacementQueryId := fmt.Sprintf(`"id":"%d"`, GetCurrMessageId(user))
				textFromFileWithNewId := reQueryId.ReplaceAllString(string(fileContent), replacementQueryId)

				replacementUserId := fmt.Sprintf(`"userId":"%s"`, user.UserId)
				textFromFileWithNewId = reUserId.ReplaceAllString(textFromFileWithNewId, replacementUserId)

				subscriptions = append(subscriptions, textFromFileWithNewId)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error walking through directory:", err)
	}

	for _, v := range subscriptions {
		user.Logger.Debugf("Sending %s", strings.ReplaceAll(v, "\n", " ")[0:60])
		//user.Logger.Debugf("Sending %s", v)
		//user.Logger.Infoln(v)
		user.WsConnectionMutex.Lock()
		err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(v))
		user.WsConnectionMutex.Unlock()
		if err != nil {
			user.Logger.Println("write 3:", err)
			return
		}
		common.AddSubscriptionSent()
		//time.Sleep(3 * time.Second)
	}
}
