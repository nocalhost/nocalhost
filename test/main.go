package main

import (
	"errors"
	"nocalhost/test/nhctl"
)

func main() {
	commitId := nhctl.GetCommitId()
	if commitId == "" {
		panic(errors.New("this is should not happen"))
	}
	nhctl.InstallNhctl(commitId)
	go nhctl.Init()
	if i := <-nhctl.StatusChan; i != 1 {
		nhctl.StopChan <- 1
	}
	defer nhctl.UninstallBookInfo()
	nhctl.InstallBookInfo()
	nhctl.PortForward()
	module := "details"
	nhctl.Dev(module)
	nhctl.Sync(module)
	nhctl.End(module)
}
