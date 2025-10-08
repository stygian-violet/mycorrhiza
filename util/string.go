package util

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode/utf8"
)

// RandomString generates a random string of the given length. It is cryptographically secure to some extent.
func RandomString(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// IsRevHash checks if the revision hash is valid.
func IsRevHash(revHash string) bool {
	if len(revHash) < 7 {
		return false
	}
	paddedRevHash := revHash
	if len(paddedRevHash)%2 != 0 {
		paddedRevHash = paddedRevHash[:len(paddedRevHash)-1]
	}
	if _, err := hex.DecodeString(paddedRevHash); err != nil {
		return false
	}
	return true
}

func Truncate(str string, maxlen int) (string, bool) {
	if utf8.RuneCountInString(str) <= max(0, maxlen) {
		return str, false
	}
	if maxlen <= 0 {
		return "", true
	}
	return string(([]rune(str))[:maxlen]), true
}

func TruncateLeft(str string, maxlen int) (string, bool) {
	count := utf8.RuneCountInString(str)
	if count <= max(0, maxlen) {
		return str, false
	}
	if maxlen <= 0 {
		return "", true
	}
	return string(([]rune(str))[count - maxlen:]), true
}

var newlineRegexp = regexp.MustCompile("\r\n?|\n\r?")

func NormalizeText(text string) string {
	text = strings.TrimSpace(text)
	text = newlineRegexp.ReplaceAllString(text, "\n")
	if text != "" {
		text += "\n"
	}
	return text
}
