package clientgoutils

import (
	"fmt"
	"os/user"
)

func getHomePath() string {
	u , err := user.Current()
	if err == nil {
		return u.HomeDir
	}
	return ""
}

func printlnErr(info string, err error) {
	fmt.Printf("%s, err: %v\n", info, err)
}
