package remote

import (
	"context"
	"crypto/md5"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	net2 "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/kubernetes"
	"net"
	"nocalhost/internal/nhctl/vpn/util"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DHCP interface {
	RentIP() net.IPNet
	ReleaseIP(ip net.IPNet)
}

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
func (d *DHCPManager) InitDHCPIfNecessary() error {
	get, err := d.client.CoreV1().ConfigMaps(d.namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	// already exists, do nothing
	if err == nil && get != nil {
		return nil
	}
	var ips []string
	for i := 2; i < 254; i++ {
		if i != 100 {
			ips = append(ips, strconv.Itoa(i))
		}
	}
	result := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.TrafficManager,
			Namespace: d.namespace,
			Labels:    map[string]string{},
		},
		Data: map[string]string{"DHCP": strings.Join(ips, ",")},
	}
	_, err = d.client.CoreV1().ConfigMaps(d.namespace).Create(context.Background(), result, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("create dhcp error, err: %v", err)
		return err
	}
	return nil
}

func (d *DHCPManager) RentIPBaseNICAddress() (*net.IPNet, error) {
	get, err := d.client.CoreV1().ConfigMaps(d.namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get ip from dhcp, err: %v", err)
		return nil, err
	}
	split := strings.Split(get.Data["DHCP"], ",")

	ip, left := getIp(split)

	get.Data["DHCP"] = strings.Join(left, ",")
	_, err = d.client.CoreV1().ConfigMaps(d.namespace).Update(context.Background(), get, metav1.UpdateOptions{})
	if err != nil {
		log.Errorf("update dhcp error after get ip, need to put ip back, err: %v", err)
		return nil, err
	}

	return &net.IPNet{
		IP:   net.IPv4(223, 254, 254, byte(ip)),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}, nil
}

func (d *DHCPManager) RentIPRandom() (*net.IPNet, error) {
	get, err := d.client.CoreV1().ConfigMaps(d.namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get ip from dhcp, err: %v", err)
		return nil, err
	}
	split := strings.Split(get.Data["DHCP"], ",")

	ip := split[0]
	split = split[1:]

	get.Data["DHCP"] = strings.Join(split, ",")
	_, err = d.client.CoreV1().ConfigMaps(d.namespace).Update(context.Background(), get, metav1.UpdateOptions{})
	if err != nil {
		log.Errorf("update dhcp error after get ip, need to put ip back, err: %v", err)
		return nil, err
	}

	atoi, _ := strconv.Atoi(ip)
	return &net.IPNet{
		IP:   net.IPv4(223, 254, 254, byte(atoi)),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}, nil
}

func getIp(availableIp []string) (int, []string) {
	var v uint32
	interfaces, _ := net.Interfaces()
	hostInterface, _ := net2.ChooseHostInterface()
out:
	for _, i := range interfaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			if hostInterface.Equal(addr.(*net.IPNet).IP) {
				hash := md5.New()
				hash.Write([]byte(i.HardwareAddr.String()))
				sum := hash.Sum(nil)
				v = util.BytesToInt(sum)
				break out
			}
		}
	}
	m := make(map[int]int)
	for _, s := range availableIp {
		atoi, _ := strconv.Atoi(s)
		m[atoi] = atoi
	}
	for {
		if k, ok := m[int(v%256)]; ok {
			delete(m, k)
			return k, getValueFromMap(m)
		} else {
			v++
		}
	}
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

func (d *DHCPManager) ReleaseIpToDHCP(ip *net.IPNet) error {
	get, err := d.client.CoreV1().ConfigMaps(d.namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get dhcp, err: %v", err)
		return err
	}
	split := strings.Split(get.Data["DHCP"], ",")
	split = append(split, strings.Split(ip.IP.To4().String(), ".")[3])
	get.Data["DHCP"] = strings.Join(sortString(split), ",")
	_, err = d.client.CoreV1().ConfigMaps(d.namespace).Update(context.Background(), get, metav1.UpdateOptions{})
	if err != nil {
		log.Errorf("update dhcp error after release ip, need to try again, err: %v", err)
		return err
	}
	return nil
}

type DHCPRecordMap struct {
	innerMap map[string]DHCPRecord
}

type DHCPRecord struct {
	Mac      string
	Ip       string
	Deadline time.Time
}

// ToDHCP Mac --> DHCPRecord
func ToDHCP(str string) (result DHCPRecordMap) {
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
			result.innerMap[split[0]] = DHCPRecord{Mac: split[0], Ip: split[1], Deadline: parse}
		}
	}
	return
}

func (maps *DHCPRecordMap) ToString() string {
	var sb strings.Builder
	for _, v := range maps.innerMap {
		sb.WriteString(fmt.Sprintf("%s#%s#%s\n", v.Mac, v.Ip, v.Deadline.String()))
	}
	return sb.String()
}

func (maps *DHCPRecordMap) GetIP() (ip string) {
	if record, ok := maps.innerMap[util.GetMacAddress().String()]; ok {
		return record.Ip
	}
	return
}

func (maps *DHCPRecordMap) RentIP(ip net.IP) *DHCPRecordMap {
	s := util.GetMacAddress().String()
	maps.innerMap[s] = DHCPRecord{
		Mac:      s,
		Ip:       ip.String(),
		Deadline: time.Now().Add(time.Second * 30),
	}
	return maps
}

// TODO rent ip daadline
func (maps *DHCPRecordMap) RentDeadline(duration time.Duration) {

}
