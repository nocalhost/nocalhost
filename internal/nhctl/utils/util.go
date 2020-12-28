package utils

import (
	"crypto/sha1"
	"fmt"
	"os/user"
)

func GetHomePath() string {
	u, err := user.Current()
	if err == nil {
		return u.HomeDir
	}
	return ""
}

func Sha1ToString(str string) string {
	hash := sha1.New()
	hash.Write([]byte(str))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
