package tpl

import (
	"fmt"
	"testing"
)

func TestGetSvcTpl(t *testing.T) {
	tpl, err := GetSvcTpl("aa")
	if err != nil {
		panic(err)
	}
	fmt.Println(tpl)
}
