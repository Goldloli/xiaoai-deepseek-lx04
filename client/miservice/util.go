package miservice

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func getRandom(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomStr := make([]byte, length)
	for i := range randomStr {
		randomStr[i] = charset[r.Intn(len(charset))]
	}
	return string(randomStr)
}

func signNonce(ssecurity string, nonce string) (string, error) {
	decodedSsecurity, err := base64.StdEncoding.DecodeString(ssecurity)
	if err != nil {
		return "", err
	}

	decodedNonce, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	hash.Write(decodedSsecurity)
	hash.Write(decodedNonce)
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

var genNonce = func() string {
	nonce := make([]byte, 12)
	_, err := rand.Read(nonce[:8])
	if err != nil {
		return ""
	}
	binary.BigEndian.PutUint32(nonce[8:], uint32(time.Now().Unix()/60))
	return base64.StdEncoding.EncodeToString(nonce)
}

func signData(uri string, data any, ssecurity string) url.Values {
	var dataStr []byte
	if s, ok := data.(string); ok {
		dataStr = []byte(s)
	} else {
		var err error
		dataStr, err = json.Marshal(data)
		if err != nil {
			return nil
		}
	}

	encodedNonce := genNonce()
	snonce, err := signNonce(ssecurity, encodedNonce)
	if err != nil {
		return nil
	}
	msg := fmt.Sprintf("%s&%s&%s&data=%s", uri, snonce, encodedNonce, dataStr)
	sb, _ := base64.StdEncoding.DecodeString(snonce)
	sign := hmac.New(sha256.New, sb)
	sign.Write([]byte(msg))
	signature := base64.StdEncoding.EncodeToString(sign.Sum(nil))
	return url.Values{
		"_nonce":    {encodedNonce},
		"data":      {string(dataStr)},
		"signature": {signature},
	}
}

func twinsSplit(str, sep string, def string) (string, string) {
	pos := strings.Index(str, sep)
	if pos == -1 {
		return str, def
	}
	return str[0:pos], str[pos+1:]
}

func stringToValue(str string) interface{} {
	switch str {
	case "null", "none":
		return nil
	case "false":
		return false
	case "true":
		return true
	default:
		if intValue, err := strconv.Atoi(str); err == nil {
			return intValue
		}
		return str
	}
}

func stringOrValue(str string) interface{} {
	if str != "" && str[0] == '#' {
		return stringToValue(str[1:])
	}
	return str
}

func unzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func isDigit(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
