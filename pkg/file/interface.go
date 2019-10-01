package file

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
)

// Entry is used to represent data stored by the physical Storage
type Entry struct {
	Key   string
	Value []byte
}

// MD5CurrentHexString -
func (e *Entry) MD5CurrentHexString() string {
	hash := md5.New()
	hash.Write(e.Value)
	md5sumCurr := hash.Sum(nil)
	var appendHyphen bool
	if len(md5sumCurr) == 0 {
		md5sumCurr = make([]byte, 16)
		rand.Read(md5sumCurr)
		appendHyphen = true
	}
	if appendHyphen {
		return hex.EncodeToString(md5sumCurr)[:32] + "-1"
	}
	return hex.EncodeToString(md5sumCurr)
}
