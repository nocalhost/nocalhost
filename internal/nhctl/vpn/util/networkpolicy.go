/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

// DeleteWindowsFirewallRule Delete all action block firewall rule
func DeleteWindowsFirewallRule() {
	_ = exec.Command("PowerShell", []string{
		"Remove-NetFirewallRule",
		"-Action",
		"Block",
	}...).Run()
}

func AddFirewallRule() {
	cmd := exec.Command("netsh", []string{
		"advfirewall",
		"firewall",
		"add",
		"rule",
		"name=" + TrafficManager,
		"dir=in",
		"action=allow",
		"enable=yes",
		fmt.Sprintf("remoteip=%s,LocalSubnet", RouterIP.String()),
	}...)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Infof("error while exec command: %s, out: %s, err: %v", cmd.Args, string(out), err)
	}
}

func FindRule() bool {
	cmd := exec.Command("netsh", []string{
		"advfirewall",
		"firewall",
		"show",
		"rule",
		"name=" + TrafficManager,
	}...)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Infof("find route out: %s error: %v", string(out), err)
		return false
	} else {
		return true
	}
}
