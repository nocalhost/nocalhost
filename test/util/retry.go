package util

import (
	"fmt"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/test/nhctlcli"
	"time"
)

var retryTimes = 10

func RetryWith3Params(funcs map[string]func(*nhctlcli.CLI, string, int) error, suiteName string, cli *nhctlcli.CLI, moduleName string, port int) {
	var err error
	for name, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(cli, moduleName, port); err == nil {
				break
			}
			time.Sleep(time.Second * 2)
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s while execute testcase: %s", suiteName, name))
	}
}

func RetryWith2Params(funcs map[string]func(*nhctlcli.CLI, string) error, suiteName string, cli *nhctlcli.CLI, moduleName string) {
	var err error
	for name, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(cli, moduleName); err == nil {
				break
			}
			time.Sleep(time.Second * 2)
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s while execute testcase: %s", suiteName, name))
	}
}

func RetryWith1Params(funcs map[string]func(*nhctlcli.CLI) error, suiteName string, cli *nhctlcli.CLI) {
	var err error
	for name, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(cli); err == nil {
				break
			}
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s while execute testcase: %s", suiteName, name))
	}
}

func RetryWithString(funcs map[string]func(string) error, suiteName string, parameter string) {
	var err error
	for name, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(parameter); err == nil {
				break
			}
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s while execute testcase: %s", suiteName, name))
	}
}
