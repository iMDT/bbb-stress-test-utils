package hasura

import (
	"bbb-stress-test/common"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"strconv"
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

	defer user.WsConnection.Close()

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

	go handleWsMessages(user)
	SendConnectionInitMessage(user)

	if user.Benchmarking {
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
			return
		}

		type Message struct {
			Type    string      `json:"type"`
			Id      string      `json:"id"`
			Payload interface{} `json:"payload"`
			//Payload map[string]interface{} `json:"payload"`
		}

		//{"id":"41",
		//"payload":{"data":{"meeting":[{"__typename":"meeting","breakoutPolicies":{"__typename":"meeting_breakoutPolicies","breakoutRooms":[],"captureNotes":false,"captureNotesFilename":"%%CONFNAME%%","captureSlides":false,"captureSlidesFilename":"%%CONFNAME%%","freeJoin":false,"parentId":"bbb-none","privateChatEnabled":true,"record":false,"sequence":0},"componentsFlags":{"__typename":"meeting_componentFlags","hasBreakoutRoom":false,"hasCaption":false,"hasExternalVideo":false,"hasPoll":false,"hasScreenshare":false,"hasTimer":false},"createdTime":1708950207625,"disabledFeatures":[],"durationInSeconds":0,"extId":"random-6403352","externalVideo":null,"html5InstanceId":null,"isBreakout":false,"lockSettings":{"__typename":"meeting_lockSettings","disableCam":false,"disableMic":false,"disableNotes":false,"disablePrivateChat":false,"disablePublicChat":false,"hasActiveLockSetting":false,"hideUserList":false,"hideViewersAnnotation":false,"hideViewersCursor":false,"webcamsOnlyForModerator":false},"maxPinnedCameras":3,"meetingCameraCap":0,"meetingId":"10bbce770f3adc35c305f0d7cc34cfc115530b5a-1708950207625","name":"random-6403352","notifyRecordingIsOn":false,"presentationUploadExternalDescription":"","presentationUploadExternalUrl":"","recordingPolicies":{"__typename":"meeting_recordingPolicies","allowStartStopRecording":true,"autoStartRecording":false,"keepEvents":false,"record":false},"screenshare":null,"usersPolicies":{"__typename":"v_meeting_usersPolicies","allowModsToEjectCameras":false,"allowModsToUnmuteUsers":false,"authenticatedGuest":true,"guestLobbyMessage":null,"guestPolicy":"ALWAYS_ACCEPT","maxUserConcurrentAccesses":3,"maxUsers":0,"meetingLayout":"CUSTOM_LAYOUT","moderatorsCanMuteAudio":true,"moderatorsCanUnmuteAudio":false,"userCameraCap":3,"webcamsOnlyForModerator":false},"voiceSettings":{"__typename":"meeting_voiceSettings","dialNumber":"613-555-1234","muteOnStart":false,"telVoice":"73939","voiceConf":"73939"}}]}},
		//"type":"data"}

		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			user.Logger.Println("error on unmarshal message:", err)
			continue
		}

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
		case "data":
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

				if messageAsMap, okMessageAsMap := msg.Payload.(map[string]interface{}); okMessageAsMap {
					if data, okData := messageAsMap["data"].(map[string]interface{}); okData {
						for _, dataItem := range data {
							if currentDataProp, okCurrentDataProp := dataItem.([]interface{}); okCurrentDataProp {
								if okCurrentDataProp && len(currentDataProp) > 0 {
									firstItemOfMessage := currentDataProp[0]
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
													SendUpdateConnectionAliveAt(user)
												}

												time.Sleep(1 * time.Second)

												if !user.Benchmarking {
													SendSubscriptionsBatch(user)
													SendChatMessages(user)
												}
											}
										}
									}
								}

							}
						}
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
		user.Logger.Println("write:", err)
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
		Type: "start",
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
		user.Logger.Errorf("error marshalling connection_init message: %v", err)
		return
	}

	user.WsConnectionMutex.Lock()
	err = user.WsConnection.WriteMessage(websocket.TextMessage, msgBytes)
	user.WsConnectionMutex.Unlock()
	if err != nil {
		user.Logger.Println("write:", err)
		return
	}
}

func SendSendGroupChatMessageMsg(user *common.User, typingMessageId int, messageId int, chatMessage string) {
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
}

func SendUpdateConnectionAliveAt(user *common.User) {
	user.ConnectionAliveMutationId = GetCurrMessageId(user)

	user.Logger.Debugf("Created alive at %d", user.ConnectionAliveMutationId)

	//Send Message
	SendGenericGraphqlMessage(
		user,
		user.ConnectionAliveMutationId,
		map[string]interface{}{},
		"UpdateConnectionAliveAt",
		`mutation UpdateConnectionAliveAt { 
													userSetConnectionAlive
												}
	`)
}

func SendUserCurrentSubscription(user *common.User) {
	user.UserCurrentSubscriptionId = GetCurrMessageId(user)

	SendGenericGraphqlMessage(
		user,
		user.UserCurrentSubscriptionId,
		make(map[string]interface{}),
		"userCurrentSubscription",
		`subscription userCurrentSubscription { user_current { authed banned joined __typename } }`)

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
			"authToken":  user.AuthToken,
			"clientType": "HTML5",
		},
		"UserJoin",
		`mutation UserJoin($authToken: String!, $clientType: String!) { 
												userJoinMeeting(authToken: $authToken, clientType: $clientType)
												}`)
}

func SendSubscriptionsBatch(user *common.User) {
	if !common.GetConfig().SendSubscriptionsBatch {
		return
	}

	user.Logger.Debugln("Sending Hasura subscriptions batch")

	//time.Sleep(1 * time.Second)

	var subscriptions []string

	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {\n  user_aggregate {\n    aggregate {\n      count\n      __typename\n    }\n    __typename\n  }\n}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {  meeting {    meetingId    isBreakout    lockSettings {      disableCam      disableMic      disableNotes      disablePrivateChat      disablePublicChat      hasActiveLockSetting      hideUserList      hideViewersCursor      webcamsOnlyForModerator      __typename    }    usersPolicies {      allowModsToEjectCameras      allowModsToUnmuteUsers      authenticatedGuest      guestPolicy      maxUserConcurrentAccesses      maxUsers      meetingLayout      userCameraCap      webcamsOnlyForModerator      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {\n  user_aggregate {\n    aggregate {\n      count\n      __typename\n    }\n    __typename\n  }\n}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {  meeting {    meetingId    isBreakout    lockSettings {      disableCam      disableMic      disableNotes      disablePrivateChat      disablePublicChat      hasActiveLockSetting      hideUserList      hideViewersCursor      webcamsOnlyForModerator      __typename    }    usersPolicies {      allowModsToEjectCameras      allowModsToUnmuteUsers      authenticatedGuest      guestPolicy      maxUserConcurrentAccesses      maxUsers      meetingLayout      userCameraCap      webcamsOnlyForModerator      __typename    }    __typename  }}"}}`)
	//subscriptions = append(subscriptions, `{"id":"33","type":"start","payload":{"variables":{},"extensions":{},"operationName":"CurrentPresentationPagesSubscription","query":"subscription CurrentPresentationPagesSubscription {  pres_page_curr {    height    isCurrentPage    num    pageId    scaledHeight    scaledViewBoxHeight    scaledViewBoxWidth    scaledWidth    svgUrl: urlsJson(path: \\"$.svg\\\`)    width    xOffset    yOffset    presentationId    content    downloadFileUri    totalPages    downloadable    presentationName    isDefaultPresentation    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {  poll {    published    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {  poll(where: {published: {_eq: true}}, order_by: [{publishedAt: desc}], limit: 1) {    ended    published    publishedAt    pollId    type    questionText    responses {      optionDesc      optionId      optionResponsesCount      pollResponsesCount      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"CursorSubscription","query":"subscription CursorSubscription {  pres_page_cursor {    isCurrentPage    lastUpdatedAt    pageId    presentationId    userId    xPercent    yPercent    user {      name      presenter      role      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{"lastUpdatedAt":"1970-01-01T00:00:00.000Z"},"extensions":{},"operationName":"annotationsStream","query":"subscription annotationsStream($lastUpdatedAt: timestamptz) {  pres_annotation_curr_stream(    batch_size: 10    cursor: {initial_value: {lastUpdatedAt: $lastUpdatedAt}}  ) {    annotationId    annotationInfo    lastUpdatedAt    pageId    presentationId    userId    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"PresentationsSubscription","query":"subscription PresentationsSubscription {  pres_presentation {    uploadInProgress    current    downloadFileUri    downloadable    uploadErrorDetailsJson    uploadErrorMsgKey    filenameConverted    isDefault    name    totalPages    totalPagesUploaded    presentationId    removable    uploadCompleted    exportToChatInProgress    exportToChatStatus    exportToChatCurrentPage    exportToChatHasError    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"PresentationsSubscription","query":"subscription PresentationsSubscription {  pres_presentation {    uploadInProgress    current    downloadFileUri    downloadable    uploadErrorDetailsJson    uploadErrorMsgKey    filenameConverted    isDefault    name    totalPages    totalPagesUploaded    presentationId    removable    uploadCompleted    __typename  }}"}}`)
	//new
	//subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"hasPendingPoll","query":"subscription hasPendingPoll($userId: String!) {  meeting {    polls(      where: {ended: {_eq: false}, users: {responded: {_eq: false}, userId: {_eq: $userId}}, userCurrent: {responded: {_eq: false}}}    ) {      users {        responded        userId        __typename      }      options {        optionDesc        optionId        pollId        __typename      }      multipleResponses      pollId      questionText      secret      type      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"getServerTime","query":"query getServerTime {  current_time {    currentTimestamp    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {  chat(    order_by: [{public: desc}, {totalUnread: desc}, {participant: {name: asc, userId: asc}}]  ) {    chatId    participant {      userId      name      role      color      loggedOut      avatar      isOnline      isModerator      __typename    }    totalMessages    totalUnread    public    lastSeenAt    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"MeetingSubscription","query":"subscription MeetingSubscription {  meeting {    createdTime    disabledFeatures    durationInSeconds    extId    lockSettings {      disableCam      disableMic      disableNotes      disablePrivateChat      disablePublicChat      hasActiveLockSetting      hideUserList      hideViewersCursor      hideViewersAnnotation      webcamsOnlyForModerator      __typename    }    maxPinnedCameras    meetingCameraCap    meetingId    name    notifyRecordingIsOn    presentationUploadExternalDescription    presentationUploadExternalUrl    recordingPolicies {      allowStartStopRecording      autoStartRecording      record      keepEvents      __typename    }    screenshare {      hasAudio      screenshareId      stream      vidHeight      vidWidth      voiceConf      screenshareConf      __typename    }    usersPolicies {      allowModsToEjectCameras      allowModsToUnmuteUsers      authenticatedGuest      guestPolicy      maxUserConcurrentAccesses      maxUsers      meetingLayout      moderatorsCanMuteAudio      moderatorsCanUnmuteAudio      userCameraCap      webcamsOnlyForModerator      guestLobbyMessage      __typename    }    isBreakout    breakoutPolicies {      breakoutRooms      captureNotes      captureNotesFilename      captureSlides      captureSlidesFilename      freeJoin      parentId      privateChatEnabled      record      sequence      __typename    }    html5InstanceId    voiceSettings {      dialNumber      muteOnStart      voiceConf      telVoice      __typename    }    externalVideo {      externalVideoId      playerCurrentTime      playerPlaybackRate      playerPlaying      externalVideoUrl      startedSharingAt      stoppedSharingAt      updatedAt      __typename    }    componentsFlags {      hasCaption      hasBreakoutRoom      hasExternalVideo      hasPoll      hasScreenshare      hasTimer      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"userCurrentSubscription","query":"subscription userCurrentSubscription {  user_current {    authed    avatar    banned    enforceLayout    cameras {      streamId      __typename    }    clientType    color    customParameters {      parameter      value      __typename    }    disconnected    away    raiseHand    emoji    extId    guest    guestStatus    hasDrawPermissionOnCurrentPage    isDialIn    isModerator    joined    lastBreakoutRoom {      breakoutRoomId      currentlyInRoom      isDefaultName      sequence      shortName      __typename    }    userClientSettings {      userClientSettingsJson      __typename    }    locked    loggedOut    mobile    name    pinned    presenter    registeredOn    role    userId    speechLocale    voice {      joined      muted      spoke      talking      listenOnly      __typename    }    __typename  }}"}}`)
	//new
	//subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"Users","query":"subscription Users($offset: Int!, $limit: Int!) {  user(    limit: $limit    offset: $offset    order_by: [{role: asc}, {raiseHandTime: asc_nulls_last}, {awayTime: asc_nulls_last}, {emojiTime: asc_nulls_last}, {isDialIn: desc}, {hasDrawPermissionOnCurrentPage: desc}, {nameSortable: asc}, {userId: asc}]  ) {    userId    extId    name    isModerator    role    color    avatar    away    raiseHand    emoji    avatar    presenter    pinned    locked    authed    mobile    guest    clientType    disconnected    loggedOut    voice {      joined      listenOnly      talking      muted      voiceUserId      __typename    }    cameras {      streamId      __typename    }    presPagesWritable {      isCurrentPage      pageId      userId      __typename    }    lastBreakoutRoom {      isDefaultName      sequence      shortName      currentlyInRoom      __typename    }    reaction {      reactionEmoji      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{"offset":0,"limit":0},"extensions":{},"operationName":"Users","query":"subscription Users($offset: Int!, $limit: Int!) {  user(    limit: $limit    offset: $offset    order_by: [{role: asc}, {raiseHandTime: asc_nulls_last}, {awayTime: asc_nulls_last}, {emojiTime: asc_nulls_last}, {isDialIn: desc}, {hasDrawPermissionOnCurrentPage: desc}, {nameSortable: asc}, {userId: asc}]  ) {    userId    extId    name    isModerator    role    color    avatar    away    raiseHand    emoji    avatar    presenter    pinned    locked    authed    mobile    guest    clientType    disconnected    loggedOut    voice {      joined      listenOnly      talking      muted      voiceUserId      __typename    }    cameras {      streamId      __typename    }    presPagesWritable {      isCurrentPage      pageId      userId      __typename    }    lastBreakoutRoom {      isDefaultName      sequence      shortName      currentlyInRoom      __typename    }    reaction {      reactionEmoji      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"getMeetingRecordingPolicies","query":"subscription getMeetingRecordingPolicies {  meeting_recordingPolicies {    allowStartStopRecording    autoStartRecording    record    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"getMeetingRecordingData","query":"subscription getMeetingRecordingData {  meeting_recording {    isRecording    startedAt    startedBy    previousRecordedTimeInSeconds    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {  user_connectionStatusReport {    user {      userId      name      avatar      color      isModerator      isOnline      __typename    }    clientNotResponding    lastUnstableStatus    lastUnstableStatusAt    currentStatus    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"subscription {  user_connectionStatus {    connectionAliveAt    userClientResponseAt    status    statusUpdatedAt    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{"limit":8},"extensions":{},"operationName":"TalkingIndicatorSubscription","query":"subscription TalkingIndicatorSubscription($limit: Int!) {  user_voice(    where: {showTalkingIndicator: {_eq: true}}    order_by: [{startTime: desc_nulls_last}, {endTime: desc_nulls_last}]    limit: $limit  ) {    callerName    spoke    talking    floor    startTime    muted    userId    user {      color      name      speechLocale      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"getIsBreakout","query":"subscription getIsBreakout {  meeting {    meetingId    isBreakout    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"MySubscription","query":"subscription MySubscription {  timer {    accumulated    active    songTrack    time    stopwatch    running    startedAt    endedAt    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"ProcessedPresentationsSubscription","query":"subscription ProcessedPresentationsSubscription {  pres_presentation(where: {uploadCompleted: {_eq: true}}) {    current    name    presentationId    __typename  }}"}}`)
	//new
	//subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"operationName":"UpdateConnectionAliveAt","query":"mutation UpdateConnectionAliveAt($userId: String, $connectionAliveAt: timestamp) {  update_user_connectionStatus(where: {}, _set: {connectionAliveAt: \\"now()\\"}) {    affected_rows    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{"chatId":"MAIN-PUBLIC-GROUP-CHAT"},"extensions":{},"operationName":"IsTyping","query":"subscription IsTyping($chatId: String!) {  user_typing_public(    order_by: {startedTypingAt: asc}    limit: 4    where: {isCurrentlyTyping: {_eq: true}, chatId: {_eq: $chatId}}  ) {    chatId    userId    isCurrentlyTyping    user {      name      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{"chatId":"MAIN-PUBLIC-GROUP-CHAT"},"extensions":{},"operationName":"GetChatData","query":"query GetChatData($chatId: String!) {  chat(where: {chatId: {_eq: $chatId}}) {    chatId    public    participant {      name      __typename    }    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{},"extensions":{},"query":"{  user_welcomeMsgs {    welcomeMsg    welcomeMsgForModerators    __typename  }}"}}`)
	subscriptions = append(subscriptions, `{"id":"`+strconv.Itoa(GetCurrMessageId(user))+`","type":"start","payload":{"variables":{"offset":0,"limit":50},"extensions":{},"operationName":"Patched_chatMessages","query":"subscription Patched_chatMessages($limit: Int!, $offset: Int!) {  chat_message_public(limit: $limit, offset: $offset, order_by: {createdAt: asc}) {    user {      name      userId      avatar      isOnline      isModerator      color      __typename    }    messageType    chatEmphasizedText    chatId    message    messageId    createdAt    messageMetadata    senderName    senderRole    __typename  }}"}}`)

	for _, v := range subscriptions {
		user.Logger.Debugf("Sending %s", v[0:30])
		//user.Logger.Infoln(v)
		user.WsConnectionMutex.Lock()
		err := user.WsConnection.WriteMessage(websocket.TextMessage, []byte(v))
		user.WsConnectionMutex.Unlock()
		if err != nil {
			user.Logger.Println("write:", err)
			return
		}
		common.AddSubscriptionSent()
	}

}
