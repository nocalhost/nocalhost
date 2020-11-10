package cmds

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewNocalHostConfig(t *testing.T) {

	config := NewNocalHostConfig("/Users/xinxinhuang/.nhctl/application/ccc/bookinfo/.nocalhost/config.yaml")
	fmt.Printf("%v\n", config)
	for _, item := range config.PreInstall {
		fmt.Println(item.Path + ":" + item.Weight)
	}

	//fmt.Printf("%v\n", config.Devs)

	for _, item := range config.Devs {
		//fmt.Printf("%s %s\n", item.Name, item.DevEnv)
		itemBytes, _ := json.Marshal(item)
		fmt.Println(string(itemBytes))
	}
}
