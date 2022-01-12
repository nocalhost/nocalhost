/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import "net"

const (
	TrafficManager string = "kubevpn.traffic.manager"
	OriginData     string = "origin_data"
	REVERSE        string = "REVERSE"
	Connect        string = "Connect"
	MacToIP        string = "MAC_TO_IP"
	DHCP           string = "DHCP"
	Splitter       string = "#"
	EndSignOK      string = "EndSignOk"
	EndSignFailed  string = "EndSignFailed"
)

var IpRange net.IP
var IpMask net.IPMask
var RouterIP net.IPNet

func init() {
	IpRange = net.IPv4(223, 254, 254, 100)
	IpMask = net.CIDRMask(24, 32)
	RouterIP = net.IPNet{IP: IpRange, Mask: IpMask}
}
