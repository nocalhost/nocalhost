package util

import "net"

const (
	TrafficManager string = "kubevpn.traffic.manager"
	OriginData     string = "origin_data"
)

var IpRange net.IP
var IpMask net.IPMask
var RouterIP net.IPNet

func init() {
	IpRange = net.IPv4(223, 254, 254, 100)
	IpMask = net.CIDRMask(24, 32)
	RouterIP = net.IPNet{IP: IpRange, Mask: IpMask}
}
