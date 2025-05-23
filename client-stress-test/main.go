package main

import (
	"bbb-stress-test/akka_apps"
	"bbb-stress-test/bbb_web"
	"bbb-stress-test/common"
	"bbb-stress-test/hasura"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var currNumOfMsgs int64 = 1

func main() {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	if len(os.Args) > 1 {
		fmt.Printf("config name: %s\n", os.Args[1])
		common.SetConfigFile(os.Args[1])
	}

	config := common.GetConfig()

	//logrus := logrus.New()
	//logger := logrus.NewEntry(logrus.New())

	logLevelFromConfig, _ := log.ParseLevel(config.LogLevel)
	log.SetLevel(logLevelFromConfig)
	//log.SetFormatter(&log.JSONFormatter{})

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
		FullTimestamp:   true,
	})

	//file, err := os.OpenFile("/tmp/benchmarking_stress.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	//if err != nil {
	//	log.Fatal("Erro ao abrir arquivo de log:", err)
	//}
	//defer file.Close()
	//log.SetOutput(file)

	log := log.WithField("_routine", "main")

	meetingId := bbb_web.RequestApiCreate(client)

	println("Meeting id: ", meetingId)

	println("")
	println("--------------------------------------------------")
	println("Use this link to join the meeting in your browser:")
	println(bbb_web.GenerateJoinUrl(meetingId, "Student 00089", "true", false))
	println("--------------------------------------------------")
	println("")

	fmt.Println(fmt.Sprintf("It will add %d users to the meeting.", config.NumOfUsers))

	log = log.WithField("meeting", meetingId)

	log.Infof("It will add %d users to the meeting.", config.NumOfUsers)

	//Start benchmarking client
	if config.BenchmarkingEnabled {
		go benchmarking(meetingId)
	}

	time.Sleep(time.Duration(config.DelayFirstUserJoinInSecs) * time.Second)

	startedAt := time.Now()

	var users []string

	for i := 0; i < config.NumOfUsers; i++ {
		name := fmt.Sprintf("Student %0*d", 5, i)
		users = append(users, name)
	}

	rand.Seed(time.Now().UnixNano())

	// Randomize list
	rand.Shuffle(len(users), func(i, j int) {
		users[i], users[j] = users[j], users[i]
	})

	for _, name := range users {
		go addNewUser(meetingId, name, false)
		delayBetweenJoins := rand.Intn(config.MaxIntervalBetweenUserJoinInMs-config.MinIntervalBetweenUserJoinInMs+1) + config.MaxIntervalBetweenUserJoinInMs
		time.Sleep(time.Duration(delayBetweenJoins) * time.Millisecond)
	}

	rand.Seed(time.Now().UnixNano())

	//log.Infof("Waiting to finish....")

	timeRunning := time.Now()

	exit := false
	for !exit {

		log.Infof("Current number of joined users::::: %d\n", common.GetNumOfJoinedUsers())

		if common.GetNumOfJoinedUsers() >= config.NumOfUsers {
			log.Infof("%d users joined! Exiting...\n", common.GetNumOfJoinedUsers())
			exit = true

			log.Infof("It took: %v seconds.\n", time.Since(startedAt).Seconds())

			time.Sleep(time.Duration(config.DelayToFinishTestSecs) * time.Second)
		}

		//Wait a benchmark user
		if config.BenchmarkingEnabled {
			time.Sleep(time.Duration(4) * time.Second)
		} else {
			time.Sleep(time.Duration(1) * time.Second)
		}

		if time.Since(timeRunning).Seconds() > float64(config.Timeout) {
			log.Infoln("Exiting due to timeout.")
			exit = true
		}
	}

	formattedDate := time.Now().Format("2006-01-02 15:04:05")

	if config.BenchmarkingEnabled {
		common.ExportCsv(formattedDate)
		common.DrawPlot(formattedDate)
	}
}

func benchmarking(meetingId string) {

	//file, err := os.OpenFile("benchmarking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	//if err != nil {
	//	log.Fatal("Erro ao abrir arquivo de log:", err)
	//}
	//defer file.Close()

	//file, err := os.OpenFile("example.txt", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)

	fileJson, err := os.OpenFile("benchmarking.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal("Error opening file:", err)
	}
	defer fileJson.Close()

	//file, _ := os.Create("benchmarking_" + meetingId + ".csv")

	//
	//logEspecial := log.New()
	//logEspecial.Out = file
	//logEspecial.SetLevel(log.InfoLevel)
	//logEspecial.SetFormatter(&log.JSONFormatter{})
	//
	//logEspecial.Info("Starting benchmarking")

	benchmarkingCurrUser := 1
	for {

		//percent, err := cpu.Percent(1*time.Second, false)
		//if err != nil {
		//	fmt.Println("Error:", err)
		//	return
		//}
		//
		//// percent is a slice of float64 values representing the usage percentages.
		//// Since we passed 'false' for percpu, it should have only one element representing the overall usage.
		//log.Infof("CPU Usage: %.2f%%\n", percent[0])
		//
		//// Get virtual memory usage statistics
		//vmStat, err := mem.VirtualMemory()
		//if err != nil {
		//	fmt.Println("Error:", err)
		//	return
		//}
		//
		//// Print some of the memory usage statistics
		//log.Infof("Memory Total: %v, Free: %v, UsedPercent: %.2f%%\n", vmStat.Total, vmStat.Free, vmStat.UsedPercent)

		//logEspecial.Info("It will add a new user-----------------")
		//logEspecial.Info("Users clients: ", common.GetNumOfUsers())
		//logEspecial.Info("Users connected: ", common.GetNumOfConnectedUsers())
		//logEspecial.Info("Users joined: ", common.GetNumOfJoinedUsers())

		go addNewUser(meetingId, fmt.Sprintf("Benchmarking %02d", benchmarkingCurrUser), true)
		benchmarkingCurrUser++
		time.Sleep(time.Duration(common.GetConfig().IntervalBetweenBenchmarkUsersInSec) * time.Second)

		if common.GetNumOfJoinedUsers() >= common.GetConfig().NumOfUsers {
			break
		}

	}
}

func addNewUser(meetingId string, name string, benchmarking bool) {

	jar, _ := cookiejar.New(nil)
	newClient := &http.Client{
		Jar: jar,
	}

	userId, sessionToken, authToken, apiCookie := bbb_web.RequestApiJoin(newClient, meetingId, name)
	log.WithField("user", name)
	//if benchmarkingLogger != nil {
	//	benchmarkingLogger = benchmarkingLogger.WithField("user", name)
	//}

	log.Debugln("sessionToken: " + sessionToken)

	if userId == "" {
		log.Errorf("It was not possible to add the user " + name)
		return
	}

	var config = common.GetConfig()

	if config.Method == "graphql" {

		user := common.User{
			UserId:             userId,
			SessionToken:       sessionToken,
			AuthToken:          authToken,
			Name:               name,
			ApiCookie:          apiCookie,
			WsConnectionClosed: true,
			ConnAckReceived:    false,
			Joined:             false,
			Pong:               false,
			Chat:               false,
			CurrMessageId:      1,
			TimeToLive:         config.UserTimeToLive,
			Logger:             log.WithField("user", name),
			Benchmarking:       benchmarking,
			//BenchmarkingLogger: benchmarkingLogger,
			BenchmarkingMetrics: make(map[string]interface{}),
			//BenchmarkingCsvWriter: benchmarkingCsvWriter,
			Problem: false,
		}

		common.AddUser()

		if user.Benchmarking {
			cmd := exec.Command("/usr/bin/docker", "stats", config.BbbDockerContainerName, "--no-stream", "--format", "{{.CPUPerc}}")
			output, err := cmd.Output()
			if err != nil {
				log.Fatal(err)
			}

			// Convert the output to a string and print it
			//log.Info(string(output))

			cpuUsage := string(output)
			cpuUsage = strings.Trim(cpuUsage, "\n")
			cpuUsage = strings.Trim(cpuUsage, "%")
			user.BenchmarkingMetrics["cpu"] = cpuUsage

			cmd = exec.Command("/usr/bin/docker", "stats", config.BbbDockerContainerName, "--no-stream", "--format", "{{.MemPerc}}")
			output, err = cmd.Output()
			if err != nil {
				log.Fatal(err)
			}

			memUsage := string(output)
			memUsage = strings.Trim(memUsage, "\n")
			memUsage = strings.Trim(memUsage, "%")
			user.BenchmarkingMetrics["mem"] = memUsage

			user.BenchmarkingMetrics["users_clients"] = common.GetNumOfUsers()
			user.BenchmarkingMetrics["users_connected"] = common.GetNumOfConnectedUsers()
			user.BenchmarkingMetrics["users_joined"] = common.GetNumOfJoinedUsers()
			user.BenchmarkingMetrics["subscriptions_sent"] = common.GetNumOfSubscriptionsSent()
			user.BenchmarkingMetrics["subscriptions_received"] = common.GetNumOfSubscriptionsReceived()
		}

		hasura.StartUser(&user)

	} else if config.Method == "redis" {
		fmt.Println("Sending Redis msg")
		akka_apps.SendValidateAuthTokenReqMsg(meetingId, userId, authToken)
		time.Sleep(1 * time.Second)
		akka_apps.SendUserJoinMeetingReqMsg(meetingId, userId, authToken)
		time.Sleep(1 * time.Second)

		if len(config.ListOfMessages) > 0 {
			numOfMessages := rand.Intn(len(config.ListOfMessages)) + 1
			for i := 0; i < numOfMessages; i++ {
				akka_apps.SendSendGroupChatMessageMsg(meetingId, userId, strconv.FormatInt(currNumOfMsgs, 10)+" "+config.ListOfMessages[i])
				currNumOfMsgs++
				time.Sleep(time.Duration(config.IntervalBetweenMessagesInMs) * time.Millisecond)
			}
		}
	}

}
