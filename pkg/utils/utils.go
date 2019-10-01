package utils

import (
	"fmt"
	"math/rand"
	"path"
	"runtime"
	"strings"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandString generate random string
func GenerateRandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		randNum := rand.Intn(len(letterBytes))
		b[i] = letterBytes[randNum]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// HasPrefix ...
func HasPrefix(s string, prefix string) bool {
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
	}
	return strings.HasPrefix(s, prefix)
}

// HasSuffix ...
func HasSuffix(s string, suffix string) bool {
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
	}
	return strings.HasSuffix(s, suffix)
}

// PathJoin ...
func PathJoin(elem ...string) string {
	trailingSlash := ""
	if len(elem) > 0 {
		if HasSuffix(elem[len(elem)-1], "/") {
			trailingSlash = "/"
		}
	}
	return path.Join(elem...) + trailingSlash
}

// PrettyPrintNumber ...
func PrettyPrintNumber(number int64) string {

	G := int64(1024 * 1024 * 1024)
	M := int64(1024 * 1024)
	K := int64(1024)

	if number > 1000*G {
		return fmt.Sprintf("%dG", number/G)
	} else if number > G {
		return fmt.Sprintf("%d,%03dM", number/(1000*M), (number/M)%1000)
	} else if number > M {
		return fmt.Sprintf("%d,%03dK", number/(1000*K), (number/K)%1000)
	} else if number > K {
		return fmt.Sprintf("%dK", number/K)
	} else {
		return fmt.Sprintf("%d", number)
	}
}

// PrettyPrintSize ...
func PrettyPrintSize(size int64) string {
	if size > 1024*1024 {
		return fmt.Sprintf("%.2fM", float64(size)/(1024.0*1024.0))
	} else if size > 1024 {
		return fmt.Sprintf("%.0fK", float64(size)/1024.0)
	} else {
		return fmt.Sprintf("%d", size)
	}
}

// PrettyPrintTime ...
func PrettyPrintTime(seconds int64) string {
	day := int64(3600 * 24)
	if seconds > day*2 {
		return fmt.Sprintf("%d days %02d:%02d:%02d",
			seconds/day, (seconds%day)/3600, (seconds%3600)/60, seconds%60)
	} else if seconds > day {
		return fmt.Sprintf("1 day %02d:%02d:%02d", (seconds%day)/3600, (seconds%3600)/60, seconds%60)
	} else if seconds > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, (seconds%3600)/60, seconds%60)
	} else {
		return "n/a"
	}
}
