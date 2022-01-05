/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dns

import (
	"fmt"
	miekgdns "github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"nocalhost/internal/nhctl/vpn/util"
	"strconv"
	"testing"
)

func TestName(t *testing.T) {
	port := util.GetAvailableUDPPortOrDie()
	fmt.Println(port)
	err := NewDNSServer("udp", "127.0.0.1:"+strconv.Itoa(port), &miekgdns.ClientConfig{
		Servers: []string{""},
		Search:  []string{"test.svc.cluster.local", "svc.cluster.local", "cluster.local"},
		Port:    "53",
		Ndots:   0,
	})
	if err != nil {
		log.Warnln(err)
	}
}
