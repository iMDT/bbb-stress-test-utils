package common

import (
	"encoding/json"
	"os"
)

type Config struct {
	AppName                            string   `json:"app_name"`
	LogLevel                           string   `json:"logLevel"`
	SecuritySalt                       string   `json:"securitySalt"`
	BbbServerHost                      string   `json:"bbbServerHost"`
	BbbDockerContainerName             string   `json:"bbbDockerContainerName"`
	NumOfUsers                         int      `json:"numOfUsers"`
	IntervalBetweenBenchmarkUsersInSec int      `json:"intervalBetweenBenchmarkUsersInSec"`
	IntervalBetweenMessagesInMs        int      `json:"intervalBetweenMessagesInMs"`
	DelayFirstUserJoinInSecs           int      `json:"delayFirstUserJoinInSecs"`
	DelayToFinishTestSecs              int      `json:"delayToFinishTestSecs"`
	MinIntervalBetweenUserJoinInMs     int      `json:"minIntervalBetweenUserJoinInMs"`
	MaxIntervalBetweenUserJoinInMs     int      `json:"maxIntervalBetweenUserJoinInMs"`
	ListOfMessages                     []string `json:"listOfMessages"`
	SendChatMessages                   bool     `json:"sendChatMessages"`
	SendSubscriptionsBatch             bool     `json:"sendSubscriptionsBatch"`
	UserTimeToLive                     int      `json:"userTimeToLive"`
	Method                             string   `json:"method"`
	Timeout                            int      `json:"timeout"`
	BenchmarkingEnabled                bool     `json:"benchmarkingEnabled"`
	Debug                              bool     `json:"debug"`
	UserJoinOrder                      string   `json:"userJoinOrder"`
}

var (
	ConfigFileName           = "config.json"
	overrideNumOfUsers       *int
	overrideSendChatMessages *bool
	overrideSecuritySalt     *string
	overrideServerHost       *string
	overrideUserJoinOrder    *string
)

func SetConfigFile(configFileName string) {
	ConfigFileName = configFileName
}

func SetNumOfUsersOverride(numOfUsers int) {
	overrideNumOfUsers = &numOfUsers
}

func SetSendChatMessagesOverride(sendChatMessages bool) {
	overrideSendChatMessages = &sendChatMessages
}

func SetSecuritySaltOverride(securitySalt string) {
	overrideSecuritySalt = &securitySalt
}

func SetServerHostOverride(serverHost string) {
	overrideServerHost = &serverHost
}

func SetUserJoinOrderOverride(userJoinOrder string) {
	overrideUserJoinOrder = &userJoinOrder
}

func GetConfig() Config {
	file, err := os.Open(ConfigFileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		panic(err)
	}

	if overrideNumOfUsers != nil {
		config.NumOfUsers = *overrideNumOfUsers
	}

	if overrideSendChatMessages != nil {
		config.SendChatMessages = *overrideSendChatMessages
	}

	if overrideSecuritySalt != nil {
		config.SecuritySalt = *overrideSecuritySalt
	}

	if overrideServerHost != nil {
		config.BbbServerHost = *overrideServerHost
	}

	if overrideUserJoinOrder != nil {
		config.UserJoinOrder = *overrideUserJoinOrder
	}

	return config
}

func GetApiUrl() string {
	// return "https://bbb27.bbbvm.imdt.com.br/bigbluebutton/api"
	config := GetConfig()
	return "https://" + config.BbbServerHost + "/bigbluebutton/api"
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
	return "wss://" + config.BbbServerHost + "/graphql"
}
