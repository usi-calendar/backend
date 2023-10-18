package utils

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const calValidator = "BEGIN:VCALENDAR"

var src = rand.NewSource(time.Now().UnixNano())

var Logger = log.Default()

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytesMaskImprSrcSB(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

func SimpleGetRequest(url *string) (*string, bool) {

	var resp *http.Response
	var err error

	resp, err = http.Get(*url)

	if err != nil {
		return nil, true
	}

	if (int)(resp.StatusCode/100) > 3 {
		return nil, true
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, true
	}

	var stringBody string = string(body)

	return &stringBody, false
}

func IsCalendarValid(cal *string) bool {
	for i := 0; i < len(calValidator); i++ {
		if (*cal)[i] != calValidator[i] {
			return false
		}
	}
	return true
}
