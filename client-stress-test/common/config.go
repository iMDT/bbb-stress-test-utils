package common

import (
	"encoding/json"
	"os"
	"sync"
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
	CheckMeetingIdMismatch             bool     `json:"checkMeetingIdMismatch"`
	ModeratorActions                   bool     `json:"moderatorActions"`
	PublishCursor                      bool     `json:"publishCursor"`
}

var (
	configFileName           = "config.json"
	overrideNumOfUsers       *int
	overrideSendChatMessages *bool
	overrideSecuritySalt     *string
	overrideServerHost       *string
	overrideUserJoinOrder    *string

	cachedConfig Config
	configOnce   sync.Once
)

// SetConfigFile must be called before the first GetConfig() call.
func SetConfigFile(name string)             { configFileName = name }
func SetNumOfUsersOverride(n int)           { overrideNumOfUsers = &n }
func SetSendChatMessagesOverride(v bool)    { overrideSendChatMessages = &v }
func SetSecuritySaltOverride(s string)      { overrideSecuritySalt = &s }
func SetServerHostOverride(h string)        { overrideServerHost = &h }
func SetUserJoinOrderOverride(order string) { overrideUserJoinOrder = &order }

// GetConfig returns the cached configuration. The config is loaded once on the
// first call; all Set*Override functions must be called before that.
func GetConfig() Config {
	configOnce.Do(loadConfig)
	return cachedConfig
}

func loadConfig() {
	f, err := os.Open(configFileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&cachedConfig); err != nil {
		panic(err)
	}

	if overrideNumOfUsers != nil       { cachedConfig.NumOfUsers = *overrideNumOfUsers }
	if overrideSendChatMessages != nil { cachedConfig.SendChatMessages = *overrideSendChatMessages }
	if overrideSecuritySalt != nil     { cachedConfig.SecuritySalt = *overrideSecuritySalt }
	if overrideServerHost != nil       { cachedConfig.BbbServerHost = *overrideServerHost }
	if overrideUserJoinOrder != nil    { cachedConfig.UserJoinOrder = *overrideUserJoinOrder }
}

func GetApiUrl() string {
	return "https://" + GetConfig().BbbServerHost + "/bigbluebutton/api"
}

func GetSalt() string {
	return GetConfig().SecuritySalt
}

func GetHasuraWs() string {
	return "wss://" + GetConfig().BbbServerHost + "/graphql"
}
