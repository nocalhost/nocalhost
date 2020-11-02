package utils

import "fmt"

func Mush(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		panic(err)
	}
}
