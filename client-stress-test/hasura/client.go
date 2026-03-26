package hasura

import (
	"bbb-stress-test/common"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// StartUser runs the full lifecycle of a simulated BBB client: HTTP warm-up
// requests, WebSocket connection, GraphQL join mutation, subscriptions, and
// optional chat messages. It blocks until the user's time-to-live expires or,
// for benchmarking users, until all measured actions complete.
func StartUser(user *common.User) {
	defer user.Logger.Info("User is leaving!")

	user.CreatedTime = time.Now()

	if user.Benchmarking {
		user.BenchmarkingMetrics["name"] = user.Name
		user.BenchmarkingMetrics["created_time"] = user.CreatedTime
		defer func() {
			user.BenchmarkingMetrics["left"] = time.Since(user.CreatedTime)
			user.BenchmarkingMetrics["left_time"] = time.Now()
			common.AddBenckmarkingUser(user)
		}()
	}

	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar}

	RequestUrlWithCookies(httpClient, user, common.GetApiUrl())
	restUrl := strings.ReplaceAll(common.GetApiUrl(), "/bigbluebutton/api", "/api/rest")
	RequestUrlWithCookies(httpClient, user, restUrl+"/meetingStaticData")
	RequestUrlWithCookies(httpClient, user, restUrl+"/userMetadata")

	for !EstablishWsConnection(user) {
		time.Sleep(5 * time.Second)
		user.Logger.Info("Trying to connect again.")
	}

	if !user.Benchmarking {
		common.AddConnectedUser()
	}
	if user.Benchmarking {
		user.BenchmarkingMetrics["connection_established"] = time.Since(user.CreatedTime)
	}

	defer func() {
		user.WsConnectionClosed = true
		user.WsConnection.Close()
	}()

	// Monitor for stalled connection and attempt reconnect.
	go func() {
		for {
			time.Sleep(20 * time.Second)
			if user.WsConnection == nil || !user.ConnAckReceived || !user.Joined {
				if user.WsConnection != nil {
					user.Logger.Info("Connection stalled, reconnecting...")
					user.Problem = true
					user.ConnAckReceived = false
					user.UserJoinMutationId = 0
					EstablishWsConnection(user)
					user.Logger.Info("Reconnected.")
					go handleWsMessages(user)
					SendConnectionInitMessage(user)
				}
			} else {
				return
			}
		}
	}()

	// Send periodic keep-alive mutations.
	go func() {
		for {
			time.Sleep(10 * time.Second)
			SendUpdateConnectionAliveAt(user, GetCurrMessageId(user))
		}
	}()

	// Benchmarking user 01 sends periodic chat messages to measure RTT.
	if user.Benchmarking && user.Name == "Benchmarking 01" {
		go func() {
			for {
				time.Sleep(1 * time.Second)
				SendPeriodicChatMessage(user, GetCurrMessageId(user))
			}
		}()
	}

	go handleWsMessages(user)
	SendConnectionInitMessage(user)

	if user.Benchmarking {
		if user.Name == "Benchmarking 01" {
			time.Sleep(999 * time.Second)
		}
		for !user.Joined || !user.Pong || !user.Chat {
			time.Sleep(1 * time.Second)
		}
	} else {
		time.Sleep(time.Duration(user.TimeToLive) * time.Second)
	}
}

// RequestUrlWithCookies performs a GET request to url, attaching the user's
// API cookies and session token. Used to prime the server-side session before
// establishing the WebSocket connection.
func RequestUrlWithCookies(client *http.Client, user *common.User, url string) {
	user.Logger.Debugln("GET", url)

	req, _ := http.NewRequest("GET", url, nil)
	for _, cookie := range user.ApiCookie {
		req.AddCookie(cookie)
	}
	req.Header.Set("x-session-token", user.SessionToken)

	resp, _ := client.Do(req)
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	user.Logger.Traceln(string(body))
}
