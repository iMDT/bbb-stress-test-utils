package common

var numOfUsers int
var numOfConnectedUsers int
var numOfJoinedUsers int
var subscriptionsSent int
var subscriptionsReceived int

func AddUser() {
	numOfUsers++
}

func GetNumOfUsers() int {
	return numOfUsers
}

func AddConnectedUser() {
	numOfConnectedUsers++
}

func GetNumOfConnectedUsers() int {
	return numOfConnectedUsers
}

func AddJoinedUser() {
	numOfJoinedUsers++
}

func GetNumOfJoinedUsers() int {
	return numOfJoinedUsers
}

func AddSubscriptionSent() {
	subscriptionsSent++
}

func GetNumOfSubscriptionsSent() int {
	return subscriptionsSent
}

func AddSubscriptionReceived() {
	subscriptionsReceived++
}

func GetNumOfSubscriptionsReceived() int {
	return subscriptionsReceived
}
