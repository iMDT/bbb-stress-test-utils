package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
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

func requestApiCreate(client *http.Client) string {
	voiceBridge := getRandomIntegerAsString()
	extMeetingId := "test-" + getRandomIntegerAsString()
	controller := "create"
	params := "attendeePW=ap&meetingID=" + extMeetingId + "&moderatorPW=mp&name=" + extMeetingId + "&voiceBridge=" + voiceBridge + "&welcome=Heeyyy"

	createUrl := getApiUrl() + "/" + controller + "?" + params + "&checksum=" + getSha1sum(controller+params+getSalt())

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

func generateJoinUrl(meetingId string, name string, redirect string, moderator bool) string {
	controller := "join"
	name = strings.Replace(name, " ", "+", -1)
	params := ""
	if moderator == true {
		params = "fullName=" + name + "&meetingID=" + meetingId + "&password=mp&redirect=" + redirect
	} else {
		params = "fullName=" + name + "&meetingID=" + meetingId + "&password=ap&redirect=" + redirect
	}

	return getApiUrl() + "/" + controller + "?" + params + "&checksum=" + getSha1sum(controller+params+getSalt())
}

func requestApiJoin(client *http.Client, meetingId string, name string) (string, string, string) {
	respJoin, err := client.Get(generateJoinUrl(meetingId, name, "false", false))
	if err != nil {
		log.Fatal(err)
	}

	defer respJoin.Body.Close()
	bodyJoin, err := ioutil.ReadAll(respJoin.Body)
	if err != nil {
		log.Fatal(err)
	}

	var objJoinResponse JoinResponse
	errJoinXmlParse := xml.Unmarshal(bodyJoin, &objJoinResponse)
	if errJoinXmlParse != nil {
		fmt.Printf("error: %v", errJoinXmlParse)
		return "", "", ""
	}

	if objJoinResponse.Returncode != "SUCCESS" {
		fmt.Println("Error on trying to create user: " + objJoinResponse.Message)
		return "", "", ""
	}
	fmt.Println("User " + objJoinResponse.UserId + " created successfully.")

	return objJoinResponse.UserId, objJoinResponse.SessionToken, objJoinResponse.AuthToken
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
	enterUrl := getApiUrl() + "/" + controller + "?sessionToken=" + sessionToken

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
