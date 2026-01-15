package common

import (
	"encoding/csv"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

type User struct {
	UserId                        string
	SessionToken                  string
	AuthToken                     string
	Name                          string
	ApiCookie                     []*http.Cookie
	WsConnection                  *websocket.Conn
	WsConnectionMutex             sync.Mutex
	WsConnectionClosed            bool
	ConnAckReceived               bool
	UserJoinMutationId            int
	UserCurrentSubscriptionId     int
	ConnectionAliveMutationId     int
	ChatMessageMutationId         int
	PeriodicChatMessageMutationId int
	PeriodicChatMessageSentAt     time.Time
	PeriodicChatMessageCounter    int
	PeriodicChatMessageRtts       []int64
	Joined                        bool
	Pong                          bool
	Chat                          bool
	CurrMessageId                 int
	TimeToLive                    int
	Logger                        *logrus.Entry
	Benchmarking                  bool
	BenchmarkingLogger            *logrus.Entry
	BenchmarkingCsvWriter         *csv.Writer
	//BenchmarkingJsonFile    *os.File
	BenchmarkingMetrics map[string]interface{}
	CreatedTime         time.Time
	Problem             bool
}
