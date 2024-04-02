package common

import (
	"crypto/sha1"
	"encoding/hex"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func GetSha1sum(input string) string {
	hasher := sha1.New()
	hasher.Write([]byte(input))
	sha1sum := hasher.Sum(nil)
	return hex.EncodeToString(sha1sum)
}

func GetRandomIntegerAsString() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Intn(900000) + 100000)
}

func GetTimestamp() int64 {
	now := time.Now()
	return now.UnixNano() / int64(time.Millisecond)
}

func GetCookieJSESSIONID(apiCookie []*http.Cookie) string {
	var cookie string
	for _, c := range apiCookie {
		if c.Name == "JSESSIONID" {
			cookie = c.String()
			break
		}
	}
	//
	//for k, c := range cookie {
	//	fmt.Println(k, c)
	//}

	return cookie
}
