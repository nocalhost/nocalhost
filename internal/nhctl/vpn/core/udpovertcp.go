/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"nocalhost/internal/nhctl/vpn/util"
	"strconv"
	"strings"
)

const (
	AddrIPv4   uint8 = 1
	AddrIPv6         = 2
	AddrDomain       = 3
)

type DatagramPacket struct {
	Type       uint8  // [1]byte
	Host       string // [?]byte, first byte is length if it's a domain
	Port       uint16 // [2]byte
	DataLength uint16 // [2]byte
	Data       []byte // []byte
}

func NewDatagramPacket(addr net.Addr, data []byte) *DatagramPacket {
	s := addr.String()
	var t uint8
	if strings.Count(s, ":") >= 2 {
		t = AddrIPv6
	} else {
		if ip := net.ParseIP(strings.Split(s, ":")[0]); ip != nil {
			t = AddrIPv4
		} else {
			t = AddrDomain
		}
	}
	host, port, _ := net.SplitHostPort(s)
	atoi, _ := strconv.Atoi(port)
	// todo if host is a domain
	return &DatagramPacket{
		Host:       host,
		Port:       uint16(atoi),
		Type:       t,
		DataLength: uint16(len(data)),
		Data:       data,
	}
}

func (addr *DatagramPacket) Addr() string {
	return net.JoinHostPort(addr.Host, strconv.Itoa(int(addr.Port)))
}

func ReadDatagramPacket(r io.Reader) (*DatagramPacket, error) {
	b := util.LPool.Get().([]byte)
	defer util.LPool.Put(b)
	_, err := io.ReadFull(r, b[:1])
	if err != nil {
		return nil, err
	}

	atype := b[0]
	d := &DatagramPacket{Type: atype}
	hostLength := 0
	switch atype {
	case AddrIPv4:
		hostLength = net.IPv4len
	case AddrIPv6:
		hostLength = net.IPv6len
	case AddrDomain:
		_, err = io.ReadFull(r, b[:1])
		if err != nil {
			return nil, err
		}
		hostLength = int(b[0])
	default:
		return nil, errors.New("")
	}

	if _, err = io.ReadFull(r, b[:hostLength]); err != nil {
		return nil, err
	}
	var host string
	switch atype {
	case AddrIPv4:
		host = net.IPv4(b[0], b[1], b[2], b[3]).String()
	case AddrIPv6:
		p := make(net.IP, net.IPv6len)
		copy(p, b[:hostLength])
		host = p.String()
	case AddrDomain:
		host = string(b[:hostLength])
	}
	d.Host = host

	if _, err = io.ReadFull(r, b[:4]); err != nil {
		return nil, err
	}
	d.Port = binary.BigEndian.Uint16(b[:2])
	d.DataLength = binary.BigEndian.Uint16(b[2:4])

	if _, err = io.ReadFull(r, b[:d.DataLength]); err != nil && (err != io.ErrUnexpectedEOF || err != io.EOF) {
		return nil, err
	}
	i := make([]byte, d.DataLength)
	copy(i, b[:d.DataLength])
	d.Data = i
	return d, nil
}

func (addr *DatagramPacket) Write(w io.Writer) error {
	buf := bytes.Buffer{}
	buf.WriteByte(addr.Type)
	switch addr.Type {
	case AddrIPv4:
		buf.Write(net.ParseIP(addr.Host).To4())
	case AddrIPv6:
		buf.Write(net.ParseIP(addr.Host).To16())
	case AddrDomain:
		buf.WriteByte(byte(len(addr.Host)))
		buf.WriteString(addr.Host)
	}
	i := make([]byte, 2)
	binary.BigEndian.PutUint16(i, addr.Port)
	buf.Write(i)
	binary.BigEndian.PutUint16(i, uint16(len(addr.Data)))
	buf.Write(i)
	if _, err := buf.Write(addr.Data); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}
