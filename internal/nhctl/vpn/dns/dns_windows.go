//go:build windows
// +build windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dns

import (
	"context"
	"fmt"
	miekgdns "github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"net"
	"os"
	"os/exec"
	"strconv"
)

func SetupDNS(config *miekgdns.ClientConfig) error {
	getenv := os.Getenv("luid")
	parseUint, err := strconv.ParseUint(getenv, 10, 64)
	if err != nil {
		log.Warningln(err)
		return err
	}
	luid := winipcfg.LUID(parseUint)
	err = luid.SetDNS(windows.AF_INET, []net.IP{net.ParseIP(config.Servers[0])}, config.Search)
	_ = exec.CommandContext(context.Background(), "ipconfig", "/flushdns").Run()
	if err != nil {
		log.Warningln(err)
		return err
	}
	//_ = updateNicMetric(tunName)
	_ = addNicSuffixSearchList(config.Search)
	return nil
}

func CancelDNS() {
	getenv := os.Getenv("luid")
	parseUint, err := strconv.ParseUint(getenv, 10, 64)
	if err != nil {
		log.Warningln(err)
		return
	}
	luid := winipcfg.LUID(parseUint)
	_ = luid.FlushDNS(windows.AF_INET)
}

func updateNicMetric(name string) error {
	cmd := exec.Command("PowerShell", []string{
		"Set-NetIPInterface",
		"-InterfaceAlias",
		fmt.Sprintf("\"%s\"", name),
		"-InterfaceMetric",
		"1",
	}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("error while update nic metrics, error: %v, output: %s, command: %v", err, string(out), cmd.Args)
	}
	return err
}

// @see https://docs.microsoft.com/en-us/powershell/module/dnsclient/set-dnsclientglobalsetting?view=windowsserver2019-ps#example-1--set-the-dns-suffix-search-list
func addNicSuffixSearchList(search []string) error {
	cmd := exec.Command("PowerShell", []string{
		"Set-DnsClientGlobalSetting",
		"-SuffixSearchList",
		fmt.Sprintf("@(\"%s\", \"%s\", \"%s\")", search[0], search[1], search[2]),
	}...)
	output, err := cmd.CombinedOutput()
	log.Info(cmd.Args)
	if err != nil {
		log.Warnf("error while set dns suffix search list, err: %v, output: %s, command: %v", err, string(output), cmd.Args)
	}
	return err
}
