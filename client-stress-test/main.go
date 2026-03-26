package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"bbb-stress-test/akka_apps"
	"bbb-stress-test/bbb_web"
	"bbb-stress-test/common"
	"bbb-stress-test/hasura"

	log "github.com/sirupsen/logrus"
)

var currNumOfMsgs int64 = 1

func main() {
	configFile := flag.String("config", "", "Path to configuration file (e.g., config.json)")
	meetingIdFlag := flag.String("meetingId", "", "Join an existing meeting instead of creating one")
	numOfUsersFlag := flag.Int("numOfUsers", -1, "Number of users (overrides config)")
	sendChatMessagesFlag := flag.Bool("sendChatMessages", false, "Send chat messages (overrides config)")
	securitySaltFlag := flag.String("securitySalt", "", "Security salt (overrides config)")
	serverHostFlag := flag.String("serverHost", "", "BBB server host (overrides config)")
	userJoinOrderFlag := flag.String("userJoinOrder", "", "Join order: asc | desc | shuffle (overrides config)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "BBB Stress Test Client\n\nUsage: %s [options]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n  %s --config=config.json\n  %s --meetingId=abc123\n\n", os.Args[0], os.Args[0])
	}

	flag.Parse()

	sendChatMessagesFlagProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "sendChatMessages" {
			sendChatMessagesFlagProvided = true
		}
	})

	// Apply flag overrides before the first GetConfig() call.
	if *configFile != "" {
		fmt.Printf("Using config: %s\n", *configFile)
		common.SetConfigFile(*configFile)
	}
	if *numOfUsersFlag != -1 {
		common.SetNumOfUsersOverride(*numOfUsersFlag)
	}
	if sendChatMessagesFlagProvided {
		common.SetSendChatMessagesOverride(*sendChatMessagesFlag)
	}
	if *securitySaltFlag != "" {
		common.SetSecuritySaltOverride(*securitySaltFlag)
	}
	if *serverHostFlag != "" {
		common.SetServerHostOverride(*serverHostFlag)
	}
	if *userJoinOrderFlag != "" {
		common.SetUserJoinOrderOverride(*userJoinOrderFlag)
	}

	config := common.GetConfig()

	logLevel, _ := log.ParseLevel(config.LogLevel)
	log.SetLevel(logLevel)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
		FullTimestamp:   true,
	})

	logger := log.WithField("_routine", "main")

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	var meetingId string
	if *meetingIdFlag != "" {
		meetingId = *meetingIdFlag
		logger.Info("Using provided meeting ID: ", meetingId)
	} else {
		meetingId = bbb_web.RequestApiCreate(client)
		logger.Info("Created meeting: ", meetingId)
	}

	fmt.Println()
	fmt.Println("--------------------------------------------------")
	fmt.Println("Join links:")
	fmt.Println()
	fmt.Println("MODERATOR:")
	fmt.Println(bbb_web.GenerateJoinUrl(meetingId, "Teacher", "true", true))
	fmt.Println()
	fmt.Println("STUDENT:")
	fmt.Println(bbb_web.GenerateJoinUrl(meetingId, "Student 00089", "true", false))
	fmt.Println("--------------------------------------------------")
	fmt.Println()

	logger = logger.WithField("meeting", meetingId)
	logger.Infof("Adding %d users to the meeting.", config.NumOfUsers)

	if config.BenchmarkingEnabled {
		go runBenchmarkingUsers(meetingId)
	}

	time.Sleep(time.Duration(config.DelayFirstUserJoinInSecs) * time.Second)

	startedAt := time.Now()
	users := buildUserList(config.NumOfUsers, config.UserJoinOrder)

	for _, name := range users {
		go addNewUser(meetingId, name, false)
		delay := rand.Intn(config.MaxIntervalBetweenUserJoinInMs-config.MinIntervalBetweenUserJoinInMs+1) + config.MinIntervalBetweenUserJoinInMs
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	timeRunning := time.Now()
	for {
		joined := common.GetNumOfJoinedUsers()
		logger.Infof("Joined users: %d", joined)

		if joined >= config.NumOfUsers {
			logger.Infof("All %d users joined in %.1f seconds.", joined, time.Since(startedAt).Seconds())
			time.Sleep(time.Duration(config.DelayToFinishTestSecs) * time.Second)
			break
		}

		if time.Since(timeRunning).Seconds() > float64(config.Timeout) {
			logger.Infoln("Exiting due to timeout.")
			break
		}

		if config.BenchmarkingEnabled {
			time.Sleep(4 * time.Second)
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	if config.BenchmarkingEnabled {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		common.ExportCsv(timestamp)
		common.DrawPlot(timestamp)
	}
}

// buildUserList creates an ordered list of user display names.
func buildUserList(n int, order string) []string {
	users := make([]string, n)
	for i := range users {
		users[i] = fmt.Sprintf("Student %0*d", 5, i)
	}

	rand.Seed(time.Now().UnixNano())

	switch strings.ToLower(strings.TrimSpace(order)) {
	case "in-order", "inorder", "forward", "asc", "ascending":
		// keep ascending order
	case "reverse", "desc", "descending":
		for i, j := 0, len(users)-1; i < j; i, j = i+1, j-1 {
			users[i], users[j] = users[j], users[i]
		}
	default: // "random", "shuffle", or unrecognised
		rand.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })
	}

	return users
}

// runBenchmarkingUsers periodically adds a benchmarking user until all normal
// users have joined.
func runBenchmarkingUsers(meetingId string) {
	fileJson, err := os.OpenFile("benchmarking.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		log.Fatal("Error opening benchmarking file:", err)
	}
	defer fileJson.Close()

	currUser := 1
	for {
		go addNewUser(meetingId, fmt.Sprintf("Benchmarking %02d", currUser), true)
		currUser++
		time.Sleep(time.Duration(common.GetConfig().IntervalBetweenBenchmarkUsersInSec) * time.Second)

		if common.GetNumOfJoinedUsers() >= common.GetConfig().NumOfUsers {
			break
		}
	}
}

func addNewUser(meetingId, name string, benchmarking bool) {
	jar, _ := cookiejar.New(nil)
	newClient := &http.Client{Jar: jar}

	userId, sessionToken, authToken, apiCookie := bbb_web.RequestApiJoin(newClient, meetingId, name)
	if userId == "" {
		log.Errorf("Could not add user %s", name)
		return
	}

	log.Debugln("sessionToken:", sessionToken)

	config := common.GetConfig()

	switch config.Method {
	case "graphql":
		user := common.User{
			UserId:              userId,
			SessionToken:        sessionToken,
			AuthToken:           authToken,
			Name:                name,
			ApiCookie:           apiCookie,
			WsConnectionClosed:  true,
			CurrMessageId:       1,
			TimeToLive:          config.UserTimeToLive,
			Logger:              log.WithField("user", name),
			Benchmarking:        benchmarking,
			BenchmarkingMetrics: make(map[string]interface{}),
		}
		common.AddUser()

		if benchmarking {
			collectDockerMetrics(&user, config)
		}

		hasura.StartUser(&user)

	case "redis":
		fmt.Println("Sending Redis msg")
		akka_apps.SendValidateAuthTokenReqMsg(meetingId, userId, authToken)
		time.Sleep(1 * time.Second)
		akka_apps.SendUserJoinMeetingReqMsg(meetingId, userId, authToken)
		time.Sleep(1 * time.Second)

		if len(config.ListOfMessages) > 0 {
			numOfMessages := rand.Intn(len(config.ListOfMessages)) + 1
			for i := 0; i < numOfMessages; i++ {
				msg := strconv.FormatInt(currNumOfMsgs, 10) + " " + config.ListOfMessages[i]
				akka_apps.SendSendGroupChatMessageMsg(meetingId, userId, msg)
				currNumOfMsgs++
				time.Sleep(time.Duration(config.IntervalBetweenMessagesInMs) * time.Millisecond)
			}
		}
	}
}

// collectDockerMetrics gathers the BBB container's CPU and memory usage along
// with current user counts into the benchmarking metrics map.
func collectDockerMetrics(user *common.User, config common.Config) {
	cpuOut, err := exec.Command("/usr/bin/docker", "stats", config.BbbDockerContainerName, "--no-stream", "--format", "{{.CPUPerc}}").Output()
	if err != nil {
		log.Fatal(err)
	}
	user.BenchmarkingMetrics["cpu"] = strings.Trim(strings.TrimRight(string(cpuOut), "\n"), "%")

	memOut, err := exec.Command("/usr/bin/docker", "stats", config.BbbDockerContainerName, "--no-stream", "--format", "{{.MemPerc}}").Output()
	if err != nil {
		log.Fatal(err)
	}
	user.BenchmarkingMetrics["mem"] = strings.Trim(strings.TrimRight(string(memOut), "\n"), "%")

	user.BenchmarkingMetrics["users_clients"] = common.GetNumOfUsers()
	user.BenchmarkingMetrics["users_connected"] = common.GetNumOfConnectedUsers()
	user.BenchmarkingMetrics["users_joined"] = common.GetNumOfJoinedUsers()
	user.BenchmarkingMetrics["subscriptions_sent"] = common.GetNumOfSubscriptionsSent()
	user.BenchmarkingMetrics["subscriptions_received"] = common.GetNumOfSubscriptionsReceived()
}
