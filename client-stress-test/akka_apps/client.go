package akka_apps

import (
	"bbb-stress-test/common"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/go-redis/redis/v8"
)

const redisChannel = "to-akka-apps-redis-channel"

var (
	redisClient *redis.Client
	redisOnce   sync.Once
)

func getRedisClient() *redis.Client {
	redisOnce.Do(func() {
		redisClient = redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		})
		if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
			log.Fatalf("Error connecting to Redis: %v", err)
		}
	})
	return redisClient
}

func publishToRedis(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Error serializing JSON message: %v", err)
	}

	result, err := getRedisClient().Publish(context.Background(), redisChannel, string(data)).Result()
	if err != nil {
		log.Fatalf("Error publishing to Redis channel: %v", err)
	}

	fmt.Printf("Message sent to channel '%s', %d subscribers received it.\n", redisChannel, result)
}

func buildMessage(name, meetingId, userId string, body interface{}) GenericReqMsg {
	return GenericReqMsg{
		Envelope: Envelope{
			Name:      name,
			Routing:   EnvelopeRouting{MeetingID: meetingId, UserID: userId},
			Timestamp: common.GetTimestamp(),
		},
		Core: Core{
			Header: Header{Name: name, MeetingID: meetingId, UserID: userId},
			Body:   body,
		},
	}
}

// --- Message types ---

type Envelope struct {
	Name      string          `json:"name"`
	Routing   EnvelopeRouting `json:"routing"`
	Timestamp int64           `json:"timestamp"`
}

type EnvelopeRouting struct {
	MeetingID string `json:"meetingId"`
	UserID    string `json:"userId"`
}

type Core struct {
	Header Header `json:"header"`
	Body   Body   `json:"body"`
}

type Header struct {
	Name      string `json:"name"`
	MeetingID string `json:"meetingId"`
	UserID    string `json:"userId"`
}

type Body interface{}

type GenericReqMsg struct {
	Envelope Envelope `json:"envelope"`
	Core     Core     `json:"core"`
}

type ValidateAuthTokenReqMsgBody struct {
	UserID    string `json:"userId"`
	AuthToken string `json:"authToken"`
}

type UserJoinMeetingReqMsgBody struct {
	UserID     string `json:"userId"`
	AuthToken  string `json:"authToken"`
	ClientType string `json:"clientType"`
}

type SendGroupChatMessageMsgBody struct {
	Msg    SendGroupChatMessageMsgBodyMsg `json:"msg"`
	ChatId string                         `json:"chatId"`
}

type SendGroupChatMessageMsgBodyMsg struct {
	CorrelationId      string                               `json:"correlationId"`
	Sender             SendGroupChatMessageMsgBodyMsgSender `json:"sender"`
	ChatEmphasizedText bool                                 `json:"chatEmphasizedText"`
	Message            string                               `json:"message"`
}

type SendGroupChatMessageMsgBodyMsgSender struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// --- Public send functions ---

func SendValidateAuthTokenReqMsg(meetingId, userId, authToken string) {
	publishToRedis(buildMessage("ValidateAuthTokenReqMsg", meetingId, userId,
		ValidateAuthTokenReqMsgBody{UserID: userId, AuthToken: authToken},
	))
}

func SendUserJoinMeetingReqMsg(meetingId, userId, authToken string) {
	publishToRedis(buildMessage("UserJoinMeetingReqMsg", meetingId, userId,
		UserJoinMeetingReqMsgBody{UserID: userId, AuthToken: authToken, ClientType: "HTML5"},
	))
}

func SendSendGroupChatMessageMsg(meetingId, userId, message string) {
	publishToRedis(buildMessage("SendGroupChatMessageMsg", meetingId, userId,
		SendGroupChatMessageMsgBody{
			Msg: SendGroupChatMessageMsgBodyMsg{
				CorrelationId:      userId + strconv.FormatInt(common.GetTimestamp(), 10),
				Sender:             SendGroupChatMessageMsgBodyMsgSender{Id: userId},
				ChatEmphasizedText: true,
				Message:            message,
			},
			ChatId: "MAIN-PUBLIC-GROUP-CHAT",
		},
	))
}
