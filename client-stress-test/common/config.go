package common

import (
	"encoding/json"
	"os"
)

type Config struct {
	AppName                            string   `json:"app_name"`
	LogLevel                           string   `json:"logLevel"`
	SecuritySalt                       string   `json:"securitySalt"`
	BbbUrl                             string   `json:"bbbUrl"`
	BbbDockerContainerName             string   `json:"bbbDockerContainerName"`
	HasuraWs                           string   `json:"hasuraWs"`
	NumOfUsers                         int      `json:"numOfUsers"`
	IntervalBetweenBenchmarkUsersInSec int      `json:"intervalBetweenBenchmarkUsersInSec"`
	IntervalBetweenMessagesInMs        int      `json:"intervalBetweenMessagesInMs"`
	DelayFirstUserJoinInSecs           int      `json:"delayFirstUserJoinInSecs"`
	MinIntervalBetweenUserJoinInMs     int      `json:"minIntervalBetweenUserJoinInMs"`
	MaxIntervalBetweenUserJoinInMs     int      `json:"maxIntervalBetweenUserJoinInMs"`
	ListOfMessages                     []string `json:"listOfMessages"`
	SendChatMessages                   bool     `json:"sendChatMessages"`
	SendSubscriptionsBatch             bool     `json:"sendSubscriptionsBatch"`
	UserTimeToLive                     int      `json:"userTimeToLive"`
	Method                             string   `json:"method"`
	Timeout                            int      `json:"timeout"`
	Debug                              bool     `json:"debug"`
}

func GetConfig() Config {
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

func GetApiUrl() string {
	//return "https://bbb27.bbbvm.imdt.com.br/bigbluebutton/api"
	config := GetConfig()
	return config.BbbUrl
}

func GetSalt() string {
	//	if len(os.Args) < 2 {
	//		fmt.Println("Use: ./bbb-stress-test [secret salt]")
	//		os.Exit(1)
	//	}
	//	return os.Args[1]

	config := GetConfig()
	return config.SecuritySalt
}

func GetHasuraWs() string {
	config := GetConfig()
	return config.HasuraWs
}
