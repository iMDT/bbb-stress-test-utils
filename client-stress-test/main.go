package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppName                        string   `json:"app_name"`
	SecuritySalt                   string   `json:"securitySalt"`
	BbbUrl                         string   `json:"bbbUrl"`
	NumOfUsers                     int      `json:"numOfusers"`
	IntervalBetweenMessagesInMs    int      `json:"intervalBetweenMessagesInMs"`
	MinIntervalBetweenUserJoinInMs int      `json:"minIntervalBetweenUserJoinInMs"`
	MaxIntervalBetweenUserJoinInMs int      `json:"maxIntervalBetweenUserJoinInMs"`
	ListOfMessages                 []string `json:"listOfMessages"`
	Debug                          bool     `json:"debug"`
}

func getConfig() Config {
	file, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		panic(err)
	}

	return config
}

func getApiUrl() string {
	//return "https://bbb27.bbbvm.imdt.com.br/bigbluebutton/api"
	config := getConfig()
	return config.BbbUrl
}

func getSalt() string {
	//	if len(os.Args) < 2 {
	//		fmt.Println("Use: ./bbb-stress-test [secret salt]")
	//		os.Exit(1)
	//	}
	//	return os.Args[1]

	config := getConfig()
	return config.SecuritySalt
}

func getSha1sum(input string) string {
	hasher := sha1.New()
	hasher.Write([]byte(input))
	sha1sum := hasher.Sum(nil)
	return hex.EncodeToString(sha1sum)
}

func getRandomIntegerAsString() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Intn(900000) + 100000)
}

func getTimestamp() int64 {
	now := time.Now()
	return now.UnixNano() / int64(time.Millisecond)
}

var currNumOfMsgs int64 = 1

func addNewUser(client *http.Client, meetingId string, name string) {

	userId, sessionToken, authToken := requestApiJoin(client, meetingId, name)
	println("sessionToken:" + sessionToken)
	//enterUser(client, sessionToken)

	if userId == "" {
		println("It was not possible to add the user " + name)
		return
	}

	time.Sleep(2 * time.Second)

	fmt.Println("Sending Redis msg")

	sendValidateAuthTokenReqMsg(meetingId, userId, authToken)
	time.Sleep(1 * time.Second)
	sendUserJoinMeetingReqMsg(meetingId, userId, authToken)
	time.Sleep(1 * time.Second)

	var config = getConfig()

	chatMessages := config.ListOfMessages
	if len(chatMessages) == 0 {
		return
	}

	rand.Seed(time.Now().UnixNano())
	numOfMessages := rand.Intn(len(chatMessages)) + 1
	for i := 0; i < numOfMessages; i++ {
		sendSendGroupChatMessageMsg(meetingId, userId, strconv.FormatInt(currNumOfMsgs, 10)+" "+chatMessages[i])
		currNumOfMsgs++
		time.Sleep(time.Duration(config.IntervalBetweenMessagesInMs) * time.Millisecond)
	}
	time.Sleep(5 * time.Second)
	for i := 0; i < numOfMessages; i++ {
		sendSendGroupChatMessageMsg(meetingId, userId, strconv.FormatInt(currNumOfMsgs, 10)+" "+chatMessages[i])
		currNumOfMsgs++
		time.Sleep(time.Duration(config.IntervalBetweenMessagesInMs) * time.Millisecond)
	}
	time.Sleep(5 * time.Second)
	for i := 0; i < numOfMessages; i++ {
		sendSendGroupChatMessageMsg(meetingId, userId, strconv.FormatInt(currNumOfMsgs, 10)+" "+chatMessages[i])
		currNumOfMsgs++
		time.Sleep(time.Duration(config.IntervalBetweenMessagesInMs) * time.Millisecond)
	}
}

func main() {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	config := getConfig()

	meetingId := requestApiCreate(client)

	println("Meeting id: ", meetingId)

	println("Use this link to join the meeting in your browser: " + generateJoinUrl(meetingId, "Teacher", "true"))

	fmt.Println(fmt.Sprintf("It will add %d users to the meeting.", config.NumOfUsers))

	time.Sleep(5 * time.Second)

	for i := 0; i < config.NumOfUsers; i++ {
		name := fmt.Sprintf("Student %0*d", 5, i)
		go addNewUser(client, meetingId, name)

		rand.Seed(time.Now().UnixNano())
		delayBetweenJoins := rand.Intn(config.MaxIntervalBetweenUserJoinInMs-config.MinIntervalBetweenUserJoinInMs+1) + config.MaxIntervalBetweenUserJoinInMs
		time.Sleep(time.Duration(delayBetweenJoins) * time.Millisecond)
	}

	println("Waiting to finish....")

	time.Sleep(1 * time.Minute)

}
