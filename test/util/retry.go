package util

import (
	"fmt"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/nhctlcli"
	"reflect"
	"runtime"
	"strings"
)

var retryTimes = 10

func RetryWith3Params(
	suiteName string, funcs []func(*nhctlcli.CLI, string, int) error, cli *nhctlcli.CLI, moduleName string, port int,
) {
	var err error
	for _, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(cli, moduleName, port); err == nil {
				break
			}
			log.Info(err)
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s - %s",
			suiteName, getFunctionName(reflect.ValueOf(f))))
	}
}

func RetryWith2Params(
	suiteName string, funcs []func(*nhctlcli.CLI, string) error, cli *nhctlcli.CLI, moduleName string,
) {
	var err error
	for _, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(cli, moduleName); err == nil {
				break
			}
			log.Info(err)
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s - %s",
			suiteName, getFunctionName(reflect.ValueOf(f))))
	}
}

func RetryWith1Params(suiteName string, funcs []func(*nhctlcli.CLI) error, cli *nhctlcli.CLI) {
	var err error
	for _, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(cli); err == nil {
				break
			}
			log.Info(err)
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s - %s",
			suiteName, getFunctionName(reflect.ValueOf(f))))
	}
}

func RetryWithString(suiteName string, funcs []func(string) error, parameter string) {
	var err error
	for _, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(parameter); err == nil {
				break
			}
			log.Info(err)
		}
		clientgoutils.MustI(err, fmt.Sprintf("error on step %s - %s",
			suiteName, getFunctionName(reflect.ValueOf(f))))
	}
}

func getFunctionName(f reflect.Value) string {
	fn := runtime.FuncForPC(f.Pointer()).Name()
	return fn[strings.LastIndex(fn, ".")+1:]
}
