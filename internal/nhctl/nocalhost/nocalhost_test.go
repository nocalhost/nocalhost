package nocalhost

import (
	"fmt"
	"nocalhost/cmd/nhctl/cmds"
	"nocalhost/pkg/nhctl/utils"
	"testing"
)

func TestNocalHost_GetApplicationDir(t *testing.T) {
	//n := NocalHost{}
	//fmt.Println(n.GetApplicationDir())
}

func TestNocalHost_GetApplications(t *testing.T) {
	n := NocalHost{}
	apps, err := n.GetApplicationNames()
	utils.Mush(err)
	for _, app := range apps {
		fmt.Println(app)
	}
}

func TestListApplications(t *testing.T) {
	cmds.ListApplications()
}
