//go:build darwin
// +build darwin

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// sudo ifconfig utun3 down
func disableDevice(conflict []string) error {
	for _, dev := range conflict {
		if err := exec.Command("sudo", "ifconfig", dev, "down").Run(); err != nil {
			return err
		}
	}
	return nil
}

func getRouteTable() (map[string][]*net.IPNet, error) {
	output, err := exec.Command("netstat", "-anr").CombinedOutput()
	if err != nil {
		return nil, err
	}
	split := strings.Split(string(output), "\n")
	routeTable := make(map[string][]*net.IPNet)
	for _, i := range split {
		fields := strings.Fields(i)
		if len(fields) >= 4 {
			cidr := fields[0]
			eth := fields[3]
			if _, ipNet, err := parseCIDR(cidr); err == nil {
				if v, ok := routeTable[eth]; ok {
					routeTable[eth] = append(v, ipNet)
				} else {
					routeTable[eth] = []*net.IPNet{ipNet}
				}
			}
		}
	}
	return routeTable, nil
}

const big = 0xFFFFFF

// Decimal to integer.
// Returns number, characters consumed, success.
func dtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return big, i, false
		}
	}
	if i == 0 {
		return 0, 0, false
	}
	return n, i, true
}

// Hexadecimal to integer.
// Returns number, characters consumed, success.
func xtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s); i++ {
		if '0' <= s[i] && s[i] <= '9' {
			n *= 16
			n += int(s[i] - '0')
		} else if 'a' <= s[i] && s[i] <= 'f' {
			n *= 16
			n += int(s[i]-'a') + 10
		} else if 'A' <= s[i] && s[i] <= 'F' {
			n *= 16
			n += int(s[i]-'A') + 10
		} else {
			break
		}
		if n >= big {
			return 0, i, false
		}
	}
	if i == 0 {
		return 0, i, false
	}
	return n, i, true
}

func parseCIDR(s string) (net.IP, *net.IPNet, error) {
	indexByte6 := strings.Count(s, ":")
	indexByte4 := strings.Count(s, ".")
	if indexByte4 < 0 && indexByte6 < 0 {
		return nil, nil, &net.ParseError{Type: "CIDR address", Text: s}
	}

	i := strings.IndexByte(s, '/')
	var addr, mask string
	if i < 0 {
		addr = s
		// ipv6
		if indexByte6 > 0 {
			mask = strconv.Itoa((indexByte6 + 1) * 8)
		} else {
			mask = strconv.Itoa((indexByte4 + 1) * 8)
		}
	} else {
		addr, mask = s[:i], s[i+1:]
	}
	iplen := net.IPv4len
	ip := parseIPv4(addr)
	if ip == nil {
		iplen = net.IPv6len
		ip = parseIPv6(addr)
	}
	n, i, ok := dtoi(mask)
	if ip == nil || !ok || i != len(mask) || n < 0 || n > 8*iplen {
		return nil, nil, &net.ParseError{Type: "CIDR address", Text: s}
	}
	m := net.CIDRMask(n, 8*iplen)
	return ip, &net.IPNet{IP: ip.Mask(m), Mask: m}, nil
}

// Parse IPv4 address (d.d.d.d).
func parseIPv4(s string) net.IP {
	var p [net.IPv4len]byte
	for i := 0; i < net.IPv4len; i++ {
		if len(s) == 0 {
			// Missing octets.
			continue
		}
		if i > 0 {
			if s[0] != '.' {
				return nil
			}
			s = s[1:]
		}
		n, c, ok := dtoi(s)
		if !ok || n > 0xFF {
			return nil
		}
		if c > 1 && s[0] == '0' {
			// Reject non-zero components with leading zeroes.
			return nil
		}
		s = s[c:]
		p[i] = byte(n)
	}
	if len(s) != 0 {
		return nil
	}
	return net.IPv4(p[0], p[1], p[2], p[3])
}

func parseIPv6(s string) (ip net.IP) {
	ip = make(net.IP, net.IPv6len)
	ellipsis := -1 // position of ellipsis in ip

	// Might have leading ellipsis
	if len(s) >= 2 && s[0] == ':' && s[1] == ':' {
		ellipsis = 0
		s = s[2:]
		// Might be only ellipsis
		if len(s) == 0 {
			return ip
		}
	}

	// Loop, parsing hex numbers followed by colon.
	i := 0
	for i < net.IPv6len {
		// Hex number.
		n, c, ok := xtoi(s)
		if !ok || n > 0xFFFF {
			i++
			continue
		}

		// If followed by dot, might be in trailing IPv4.
		if c < len(s) && s[c] == '.' {
			if ellipsis < 0 && i != net.IPv6len-net.IPv4len {
				// Not the right place.
				return nil
			}
			if i+net.IPv4len > net.IPv6len {
				// Not enough room.
				return nil
			}
			ip4 := parseIPv4(s)
			if ip4 == nil {
				return nil
			}
			ip[i] = ip4[12]
			ip[i+1] = ip4[13]
			ip[i+2] = ip4[14]
			ip[i+3] = ip4[15]
			s = ""
			i += net.IPv4len
			break
		}

		// Save this 16-bit chunk.
		ip[i] = byte(n >> 8)
		ip[i+1] = byte(n)
		i += 2

		// Stop at end of string.
		s = s[c:]
		if len(s) == 0 {
			break
		}

		// Otherwise must be followed by colon and more.
		if s[0] != ':' || len(s) == 1 {
			return nil
		}
		s = s[1:]

		// Look for ellipsis.
		if s[0] == ':' {
			if ellipsis >= 0 { // already have one
				return nil
			}
			ellipsis = i
			s = s[1:]
			if len(s) == 0 { // can be at end
				break
			}
		}
	}

	// Must have used entire string.
	if len(s) != 0 {
		return nil
	}

	// If didn't parse enough, expand ellipsis.
	if i < net.IPv6len {
		if ellipsis < 0 {
			return nil
		}
		n := net.IPv6len - i
		for j := i - 1; j >= ellipsis; j-- {
			ip[j+n] = ip[j]
		}
		for j := ellipsis + n - 1; j >= ellipsis; j-- {
			ip[j] = 0
		}
	} else if ellipsis >= 0 {
		// Ellipsis must represent at least one 0 group.
		return nil
	}
	return ip
}
