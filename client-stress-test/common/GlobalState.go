package common

import "sync/atomic"

var (
	numOfUsers            int64
	numOfConnectedUsers   int64
	numOfJoinedUsers      int64
	subscriptionsSent     int64
	subscriptionsReceived int64
)

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
