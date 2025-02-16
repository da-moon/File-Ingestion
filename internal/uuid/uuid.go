package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/palantir/stacktrace"
)

// GenerateRandomBytes is used to generate random bytes of given size.
func GenerateRandomBytes(size int) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return nil, stacktrace.NewError("failed to read random bytes: %v", err)
	}
	return buf, nil
}

const uuidLen = 16

// GenerateUUID is used to generate a random UUID
func GenerateUUID() (string, error) {
	buf, err := GenerateRandomBytes(uuidLen)
	if err != nil {
		return "", err
	}
	return FormatUUID(buf)
}

// FormatUUID -
func FormatUUID(buf []byte) (string, error) {
	if buflen := len(buf); buflen != uuidLen {
		return "", stacktrace.NewError("wrong length byte slice (%d)", buflen)
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16]), nil
}

// ParseUUID -
func ParseUUID(uuid string) ([]byte, error) {
	if len(uuid) != 2*uuidLen+4 {
		return nil, stacktrace.NewError("uuid string is wrong length")
	}

	if uuid[8] != '-' ||
		uuid[13] != '-' ||
		uuid[18] != '-' ||
		uuid[23] != '-' {
		return nil, stacktrace.NewError("uuid is improperly formatted")
	}

	hexStr := uuid[0:8] + uuid[9:13] + uuid[14:18] + uuid[19:23] + uuid[24:36]

	ret, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	if len(ret) != uuidLen {
		return nil, stacktrace.NewError("decoded hex is the wrong length")
	}

	return ret, nil
}
