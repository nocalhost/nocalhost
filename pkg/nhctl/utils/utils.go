package utils

import "fmt"

func Mush(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		panic(err)
	}
}

func MushWithErrInfo(info string, err error) {
	if err != nil {
		fmt.Printf("%s, err : %v\n", info, err)
		panic(err)
	}
}
