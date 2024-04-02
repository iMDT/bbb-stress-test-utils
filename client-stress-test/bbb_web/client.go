package bbb_web

import (
	"bbb-stress-test/common"
	"encoding/json"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
)

type CreateResponse struct {
	XMLName           xml.Name `xml:"response"`
	Returncode        string   `xml:"returncode"`
	Message           string   `xml:"message"`
	MeetingID         string   `xml:"meetingID"`
	InternalMeetingID string   `xml:"internalMeetingID"`
}

func RequestApiCreate(client *http.Client) string {
	log.WithField("_routine", "bbb_web_client")

	voiceBridge := common.GetRandomIntegerAsString()
	extMeetingId := "test-" + common.GetRandomIntegerAsString()
	controller := "create"
	params := "attendeePW=ap&meetingID=" + extMeetingId + "&moderatorPW=mp&name=" + extMeetingId + "&voiceBridge=" + voiceBridge + "&welcome=Heeyyy"

	createUrl := common.GetApiUrl() + "/" + controller + "?" + params + "&checksum=" + common.GetSha1sum(controller+params+common.GetSalt())

	//https://bbb30.bbb.imdt.dev/bigbluebutton/api/create?allowStartStopRecording=true&attendeePW=ap&autoStartRecording=false&meetingID=random-7594033&moderatorPW=mp&name=meeting+test&record=false&voiceBridge=79995&welcome=%3Cbr%3EWelcome+to+%3Cb%3E%25%25CONFNAME%25%25%3C%2Fb%3E%21&checksum=b36c0b51b1408ec4948244a22d2e2c805209b729

	//log.Info(createUrl)

	respCreate, err := client.Get(createUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer respCreate.Body.Close()
	bodyCreate, err := ioutil.ReadAll(respCreate.Body)
	if err != nil {
		log.Fatal(err)
	}

	var objCreateResponse CreateResponse
	errCreateXmlParse := xml.Unmarshal(bodyCreate, &objCreateResponse)
	if errCreateXmlParse != nil {
		fmt.Printf("error: %v", errCreateXmlParse)
		return ""
	}

	if objCreateResponse.Returncode != "SUCCESS" {
		fmt.Println("Error on trying to create meeting: " + objCreateResponse.Message)
		return ""
	}
	fmt.Println("Meeting " + objCreateResponse.MeetingID + " created successfully.")

	return objCreateResponse.InternalMeetingID
}

type JoinResponse struct {
	XMLName      xml.Name `xml:"response"`
	Returncode   string   `xml:"returncode"`
	Message      string   `xml:"message"`
	UserId       string   `xml:"user_id"`
	AuthToken    string   `xml:"auth_token"`
	SessionToken string   `xml:"session_token"`
	Url          string   `xml:"url"`
}

func GenerateJoinUrl(meetingId string, name string, redirect string, moderator bool) string {
	controller := "join"
	name = strings.Replace(name, " ", "+", -1)
	params := ""
	if moderator == true {
		params = "fullName=" + name + "&meetingID=" + meetingId + "&password=mp&redirect=" + redirect
	} else {
		params = "fullName=" + name + "&meetingID=" + meetingId + "&password=ap&redirect=" + redirect
	}

	return common.GetApiUrl() + "/" + controller + "?" + params + "&checksum=" + common.GetSha1sum(controller+params+common.GetSalt())
}

func RequestApiJoin(client *http.Client, meetingId string, name string) (string, string, string, []*http.Cookie) {
	//log := log.WithField("_routine", "bbb_web_client")

	respJoin, err := client.Get(GenerateJoinUrl(meetingId, name, "false", false))
	if err != nil {
		log.WithField("user", name).Fatal(err)
	}

	defer respJoin.Body.Close()
	bodyJoin, err := ioutil.ReadAll(respJoin.Body)
	if err != nil {
		log.WithField("user", name).Fatal(err)
	}

	var objJoinResponse JoinResponse
	errJoinXmlParse := xml.Unmarshal(bodyJoin, &objJoinResponse)
	if errJoinXmlParse != nil {
		fmt.Printf("error: %v", errJoinXmlParse)
		return "", "", "", nil
	}

	if objJoinResponse.Returncode != "SUCCESS" {
		log.WithField("user", name).Errorln("Error on trying to create user: " + objJoinResponse.Message)
		return "", "", "", nil
	}
	log.WithField("user", name).Debugln("Created successfully.")

	return objJoinResponse.UserId, objJoinResponse.SessionToken, objJoinResponse.AuthToken, respJoin.Cookies()
}

type ResponseApiEnterData struct {
	ReturnCode string `json:"returncode"`
	Message    string `json:"message"`
	MessageKey string `json:"messageKey"`
	LogoutURL  string `json:"logoutURL"`
	Fullname   string `json:"fullname"`
}

type ApiEnterResponse struct {
	Response ResponseApiEnterData `json:"response"`
}

func requestApiEnter(client *http.Client, sessionToken string) {
	controller := "enter"
	enterUrl := common.GetApiUrl() + "/" + controller + "?sessionToken=" + sessionToken

	resp, err := client.Get(enterUrl)
	if err != nil {
		log.Fatalf("Erro ao fazer a requisição: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Erro ao ler o corpo da resposta: %v", err)
	}

	var apiResponse ApiEnterResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Fatalf("Erro ao decodificar o JSON: %v", err)
	}

	fmt.Printf("ReturnCode: %s\n", apiResponse.Response.ReturnCode)

	if apiResponse.Response.ReturnCode != "SUCCESS" {
		fmt.Printf("Message: %s\n", apiResponse.Response.Message)
		return
	}

	fmt.Printf("Fullname: %s\n", apiResponse.Response.Fullname)
}
