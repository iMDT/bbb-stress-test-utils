package common

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	numOfUsers            int64
	numOfConnectedUsers   int64
	numOfJoinedUsers      int64
	subscriptionsSent     int64
	subscriptionsReceived int64
)

var (
	mismatchLogOnce sync.Once
	mismatchLogFile *os.File
	mismatchLogMu   sync.Mutex
)

// LogMeetingIdMismatch writes a meetingId mismatch event to a dedicated log
// file named after the app_name config field. Using a separate file allows
// multiple parallel stress-test runs to be inspected independently.
func LogMeetingIdMismatch(userId, operationName, dataKey, got, expected string) {
	mismatchLogOnce.Do(func() {
		appName := GetConfig().AppName
		if appName == "" {
			appName = "stress-test"
		}
		f, err := os.OpenFile(
			fmt.Sprintf("meetingid-mismatch-%s.log", appName),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open meetingId mismatch log: %v\n", err)
			return
		}
		mismatchLogFile = f
	})

	if mismatchLogFile == nil {
		return
	}

	line := fmt.Sprintf("%s [%s] meetingId mismatch in %s (user=%s): got %q, expected %q\n",
		time.Now().Format(time.RFC3339), operationName, dataKey, userId, got, expected,
	)
	mismatchLogMu.Lock()
	_, _ = mismatchLogFile.WriteString(line)
	mismatchLogMu.Unlock()
}

func AddUser()                       { atomic.AddInt64(&numOfUsers, 1) }
func GetNumOfUsers() int             { return int(atomic.LoadInt64(&numOfUsers)) }

func AddConnectedUser()              { atomic.AddInt64(&numOfConnectedUsers, 1) }
func GetNumOfConnectedUsers() int    { return int(atomic.LoadInt64(&numOfConnectedUsers)) }

func AddJoinedUser()                 { atomic.AddInt64(&numOfJoinedUsers, 1) }
func GetNumOfJoinedUsers() int       { return int(atomic.LoadInt64(&numOfJoinedUsers)) }

func AddSubscriptionSent()           { atomic.AddInt64(&subscriptionsSent, 1) }
func GetNumOfSubscriptionsSent() int { return int(atomic.LoadInt64(&subscriptionsSent)) }

func AddSubscriptionReceived()           { atomic.AddInt64(&subscriptionsReceived, 1) }
func GetNumOfSubscriptionsReceived() int { return int(atomic.LoadInt64(&subscriptionsReceived)) }

