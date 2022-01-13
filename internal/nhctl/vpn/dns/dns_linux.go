//go:build linux
// +build linux

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dns

import (
	miekgdns "github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

// systemd-resolve --status, systemd-resolve --flush-caches
func SetupDNS(config *miekgdns.ClientConfig) error {
	tunName := os.Getenv("tunName")
	if len(tunName) == 0 {
		tunName = "tun0"
	}
	cmd := exec.Command("systemd-resolve", []string{
		"--set-dns",
		config.Servers[0],
		"--interface",
		tunName,
		"--set-domain=" + config.Search[0],
		"--set-domain=" + config.Search[1],
		"--set-domain=" + config.Search[2],
	}...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("cmd: %s, output: %s, error: %v\n", cmd.Args, string(output), err)
	}

	return nil
}

func CancelDNS() {
}
