/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package remote

import (
	"context"
	"crypto/md5"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"net"
	"nocalhost/internal/nhctl/vpn/util"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DHCPManager struct {
	client    *kubernetes.Clientset
	namespace string
	cidr      *net.IPNet
}

func NewDHCPManager(client *kubernetes.Clientset, namespace string, addr *net.IPNet) *DHCPManager {
	return &DHCPManager{
		client:    client,
		namespace: namespace,
		cidr:      addr,
	}
}

//	todo optimize dhcp, using mac address, ip and deadline as unit
func (d *DHCPManager) InitDHCPIfNecessary(ctx context.Context) (*v1.ConfigMap, error) {
	configMap, err := d.client.CoreV1().ConfigMaps(d.namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	// already exists, do nothing
	if err == nil && configMap != nil {
		return configMap, nil
	}

	result := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.TrafficManager,
			Namespace: d.namespace,
			Labels:    map[string]string{},
		},
		Data: map[string]string{util.DHCP: ToString(map[string]sets.Int{
			util.GetMacAddress().String(): sets.NewInt(100),
		})},
	}
	configMap, err = d.client.CoreV1().ConfigMaps(d.namespace).Create(context.Background(), result, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		util.GetLoggerFromContext(ctx).Errorf("create DHCP error, err: %v", err)
		return nil, err
	}
	return configMap, nil
}

// ToString mac address --> rent ips
func ToString(m map[string]sets.Int) string {
	sb := strings.Builder{}
	for mac, ips := range m {
		strSet := sets.NewString()
		for _, i := range ips.List() {
			strSet.Insert(strconv.Itoa(i))
		}
		sb.WriteString(fmt.Sprintf("%s%s%s\\n", mac, util.Splitter, strings.Join(strSet.List(), ",")))
	}
	return sb.String()
}

func FromStringToDHCP(str string) map[string]sets.Int {
	var result = make(map[string]sets.Int)
	for _, line := range strings.Split(str, "\n") {
		if split := strings.Split(line, util.Splitter); len(split) == 2 {
			ints := sets.NewInt()
			for _, s := range strings.Split(split[1], ",") {
				if atoi, err := strconv.Atoi(s); err == nil {
					ints.Insert(atoi)
				}
			}
			result[split[0]] = ints
		}
	}
	return result
}

func GetAvailableIPs(m map[string]sets.Int) sets.Int {
	used := sets.NewInt()
	for _, s := range m {
		used.Insert(s.List()...)
	}
	available := sets.NewInt()
	// network mask is 24, so available ip is from 223.254.254.2 - 223.254.254.254
	for i := 2; i < 254; i++ {
		// 223.254.254.100 is reserved ip
		if !used.Has(i) && i != 100 {
			available.Insert(i)
		}
	}
	return available
}

func (d *DHCPManager) RentIP(random bool) (*net.IPNet, error) {
	configMap, err := d.client.CoreV1().ConfigMaps(d.namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get ip from dhcp, err: %v", err)
		return nil, err
	}
	//split := strings.Split(get.Data["DHCP"], ",")
	used := FromStringToDHCP(configMap.Data[util.DHCP])
	ps := GetAvailableIPs(used)
	var ip int
	if random {
		var ok bool
		if ip, ok = ps.PopAny(); !ok {
			log.Errorf("cat not get ip from dhcp, no available ip")
		}
	} else {
		ip = getIP(GetAvailableIPs(used))
	}
	if v, found := used[util.GetMacAddress().String()]; found {
		v.Insert(ip)
	} else {
		used[util.GetMacAddress().String()] = sets.NewInt(ip)
	}

	_, err = d.client.CoreV1().ConfigMaps(d.namespace).Patch(
		context.Background(),
		configMap.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf("{\"data\":{\"%s\":\"%s\"}}", util.DHCP, ToString(used))),
		metav1.PatchOptions{})
	if err != nil {
		log.Errorf("update dhcp error after get ip, need to put ip back, err: %v", err)
		return nil, err
	}

	return &net.IPNet{
		IP:   net.IPv4(223, 254, 254, byte(ip)),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}, nil
}

// get ip base on Mac address
func getIP(availableIp sets.Int) int {
	hash := md5.New()
	hash.Write([]byte(util.GetMacAddress().String()))
	sum := hash.Sum(nil)
	v := util.BytesToInt(sum)
	for retry := 1; retry < 255; retry++ {
		if i := int(v % 255); availableIp.Has(i) {
			return i
		}
		v++
	}
	return int(util.BytesToInt(sum) % 255)
}

func getValueFromMap(m map[int]int) []string {
	var result []int
	for _, v := range m {
		result = append(result, v)
	}
	sort.Ints(result)
	var s []string
	for _, i := range result {
		s = append(s, strconv.Itoa(i))
	}
	return s
}

func sortString(m []string) []string {
	var result []int
	for _, v := range m {
		if len(v) > 0 {
			if atoi, err := strconv.Atoi(v); err == nil {
				result = append(result, atoi)
			}
		}
	}
	sort.Ints(result)
	var s []string
	for _, i := range result {
		s = append(s, strconv.Itoa(i))
	}
	return s
}

func (d *DHCPManager) ReleaseIP(ips ...int) error {
	configMap, err := d.client.CoreV1().ConfigMaps(d.namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	used := FromStringToDHCP(configMap.Data[util.DHCP])
	for _, ip := range ips {
		for k := range used {
			//used[k].Delete(int(ip.IP.To4()[3]))
			used[k].Delete(ip)
		}
	}
	configMap.Data[util.DHCP] = ToString(used)
	_, err = d.client.CoreV1().ConfigMaps(d.namespace).Update(context.Background(), configMap, metav1.UpdateOptions{})
	return err
}

type DHCPRecordMap struct {
	innerMap map[string]DHCPRecord
}

//func (maps DHCPRecordMap) MacToIP() map[string]string {
//	result := make(map[string]string)
//	for _, record := range maps.innerMap {
//		result[record.Mac] = record.IP
//	}
//	return result
//}

type DHCPRecord struct {
	Mac      string
	IP       string
	Deadline time.Time
}

// FromStringToMac2IP Mac --> DHCPRecord
func FromStringToMac2IP(str string) (result DHCPRecordMap) {
	result.innerMap = map[string]DHCPRecord{}
	for _, s := range strings.Split(str, "\n") {
		// mac:ip:deadline
		split := strings.Split(s, "#")
		if len(split) == 3 {
			parse, err := time.Parse(time.RFC3339, split[2])
			if err != nil {
				// default deadline is 30min
				parse = time.Now().Add(time.Minute * 30)
			}
			result.innerMap[split[0]] = DHCPRecord{Mac: split[0], IP: split[1], Deadline: parse}
		}
	}
	return
}

func (maps *DHCPRecordMap) ToString() string {
	var sb strings.Builder
	for _, v := range maps.innerMap {
		sb.WriteString(strings.Join([]string{v.Mac, v.IP, v.Deadline.Format(time.RFC3339)}, util.Splitter) + "\\n")
	}
	return sb.String()
}

func (maps *DHCPRecordMap) ToMac2IPMap() map[string]string {
	var result = make(map[string]string)
	for _, record := range maps.innerMap {
		result[record.Mac] = record.IP
	}
	return result
}

func (maps *DHCPRecordMap) GetIPByMac(mac string) (ip string) {
	if record, ok := maps.innerMap[mac]; ok {
		return record.IP
	}
	return
}

func (maps *DHCPRecordMap) AddMacToIPRecord(mac string, ip net.IP) *DHCPRecordMap {
	maps.innerMap[mac] = DHCPRecord{
		Mac:      mac,
		IP:       ip.String(),
		Deadline: time.Now().Add(time.Second * 30),
	}
	return maps
}
