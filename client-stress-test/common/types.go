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
	IsModerator         bool
	SeenUserIds         map[string]bool
	MeetingId           string
	SubscriptionNames    map[int]string            // subscription message ID → GraphQL operationName
	SubscriptionLastData map[int]map[string][]byte // subscription message ID → last known full payload.data per key

	CurrentPresentationPagesSubscriptionId int    // msgId of CurrentPresentationPagesSubscription, used to detect when pageId data arrives
	CurrentPageId                          string // pageId currently used in the active pageId-dependent subscriptions
	PresPageCurrState                      []byte // last known pres_page_curr JSON array, kept in sync by applying patches
	ActivePageIdSubMsgIds                  []int  // msgIds of currently active pageId-dependent subscriptions; used to cancel on page change
}
