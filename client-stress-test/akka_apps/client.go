package akka_apps

import (
	"bbb-stress-test/common"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"strconv"
)

import (
	"context"
)

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

//type ValidateAuthTokenReqMsg struct {
//	Envelope Envelope `json:"envelope"`
//	Core     Core     `json:"core"`
//}

type ValidateAuthTokenReqMsgBody struct {
	UserID    string `json:"userId"`
	AuthToken string `json:"authToken"`
}

//
//type UserJoinMeetingReqMsg struct {
//	Envelope Envelope `json:"envelope"`
//	Core     Core     `json:"core"`
//}

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

func SendValidateAuthTokenReqMsg(meetingId string, userId string, authToken string) {
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// Testar a conexão com o servidor Redis
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis server: %v", err)
	}

	// Criar a mensagem JSON
	messageData := GenericReqMsg{
		Envelope: Envelope{
			Name:      "ValidateAuthTokenReqMsg",
			Routing:   EnvelopeRouting{MeetingID: meetingId, UserID: userId},
			Timestamp: common.GetTimestamp(),
		},
		Core: Core{
			Header: Header{
				Name:      "ValidateAuthTokenReqMsg",
				MeetingID: meetingId,
				UserID:    userId,
			},
			Body: ValidateAuthTokenReqMsgBody{
				UserID:    userId,
				AuthToken: authToken,
			},
		},
	}

	// Serializar a mensagem JSON
	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		log.Fatalf("Error serializing JSON message: %v", err)
	}

	// Enviar a mensagem JSON para o canal do Redis
	channel := "to-akka-apps-redis-channel"
	pubResult, err := client.Publish(ctx, channel, string(messageJSON)).Result()
	if err != nil {
		log.Fatalf("Error sending message to Redis channel: %v", err)
	}

	fmt.Printf("JSON message sent to channel '%s', %d subscribers received the message.\n", channel, pubResult)

	// Fechar o cliente Redis
	client.Close()
}

func SendUserJoinMeetingReqMsg(meetingId string, userId string, authToken string) {
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// Testar a conexão com o servidor Redis
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis server: %v", err)
	}

	// Criar a mensagem JSON
	messageData := GenericReqMsg{
		Envelope: Envelope{
			Name:      "UserJoinMeetingReqMsg",
			Routing:   EnvelopeRouting{MeetingID: meetingId, UserID: userId},
			Timestamp: common.GetTimestamp(),
		},
		Core: Core{
			Header: Header{
				Name:      "UserJoinMeetingReqMsg",
				MeetingID: meetingId,
				UserID:    userId,
			},
			Body: UserJoinMeetingReqMsgBody{
				UserID:     userId,
				AuthToken:  authToken,
				ClientType: "HTML5",
			},
		},
	}

	// Serializar a mensagem JSON
	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		log.Fatalf("Error serializing JSON message: %v", err)
	}

	// Enviar a mensagem JSON para o canal do Redis
	channel := "to-akka-apps-redis-channel"
	pubResult, err := client.Publish(ctx, channel, string(messageJSON)).Result()
	if err != nil {
		log.Fatalf("Error sending message to Redis channel: %v", err)
	}

	fmt.Printf("JSON message sent to channel '%s', %d subscribers received the message.\n", channel, pubResult)

	// Fechar o cliente Redis
	client.Close()
}

func SendSendGroupChatMessageMsg(meetingId string, userId string, message string) {
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// Testar a conexão com o servidor Redis
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis server: %v", err)
	}

	// Criar a mensagem JSON
	messageData := GenericReqMsg{
		Envelope: Envelope{
			Name:      "SendGroupChatMessageMsg",
			Routing:   EnvelopeRouting{MeetingID: meetingId, UserID: userId},
			Timestamp: common.GetTimestamp(),
		},
		Core: Core{
			Header: Header{
				Name:      "SendGroupChatMessageMsg",
				MeetingID: meetingId,
				UserID:    userId,
			},
			Body: SendGroupChatMessageMsgBody{
				Msg: SendGroupChatMessageMsgBodyMsg{
					CorrelationId:      userId + strconv.FormatInt(common.GetTimestamp(), 10),
					Sender:             SendGroupChatMessageMsgBodyMsgSender{Id: userId, Name: "", Role: ""},
					ChatEmphasizedText: true,
					Message:            message,
				},
				ChatId: "MAIN-PUBLIC-GROUP-CHAT",
			},
		},
	}

	type SendGroupChatMessageMsgBody struct {
		Msg    SendGroupChatMessageMsgBodyMsg `json:"correlationId"`
		chatId string                         `json:"chatId"`
	}

	type SendGroupChatMessageMsgBodyMsg struct {
		CorrelationId      string                               `json:"correlationId"`
		sender             SendGroupChatMessageMsgBodyMsgSender `json:"sender"`
		chatEmphasizedText string                               `json:"chatEmphasizedText"`
		message            string                               `json:"message"`
	}

	type SendGroupChatMessageMsgBodyMsgSender struct {
		id   string `json:"id"`
		name string `json:"name"`
		role string `json:"role"`
	}

	// Serializar a mensagem JSON
	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		log.Fatalf("Error serializing JSON message: %v", err)
	}

	// Enviar a mensagem JSON para o canal do Redis
	channel := "to-akka-apps-redis-channel"
	pubResult, err := client.Publish(ctx, channel, string(messageJSON)).Result()
	if err != nil {
		log.Fatalf("Error sending message to Redis channel: %v", err)
	}

	fmt.Printf("JSON message sent to channel '%s', %d subscribers received the message.\n", channel, pubResult)

	// Fechar o cliente Redis
	client.Close()
}
