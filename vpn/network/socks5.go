package network

import (
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/test/nhctlcli"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// todo dumpServiceToHostsFile or find another way to solve dns issue
func sshD(privateKeyPath string, localSshPort int) {
	sock5Port, _ := ports.GetAvailablePort()
	_, _ = proxy.SOCKS5("", "", nil, nil)
	_ = "ssh -ND 0.0.0.0:8080 root@127.0.0.1 -p 5000 -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -i /Users/naison/.nh/ssh/private/key"
	_ = "curl --socks5 127.0.0.1:8080 172.16.255.110:8080"
	_ = "export http_proxy=socks5://127.0.0.1:8080"
	cmd := exec.Command("ssh", "-ND",
		"0.0.0.0:"+strconv.Itoa(sock5Port),
		"root@127.0.0.1",
		"-p", strconv.Itoa(localSshPort),
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-i", privateKeyPath)
	go func() {
		stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
		if err != nil {
			log.Printf("ssh -d err: %v, stdout: %s, stderr: %s\n", err, stdout, stderr)
		}
	}()
	time.Sleep(time.Second * 3)
	dumpServiceToHosts()
	fmt.Printf(`please export http_proxy=socks5://127.0.0.1:%d, and the you can access cluster ip or domain`, sock5Port)
}

func dumpServiceToHosts() {
	log.Println("prepare to dump service to hosts")
	list, err2 := clientset.CoreV1().Services(v1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err2 != nil {
		log.Println("error while list all namespace service")
	}

	_, err2 = CopyFile("/etc/hosts.bak", "/etc/hosts")
	if err2 != nil {
		log.Printf("backup hosts file failed")
	} else {
		log.Printf("backup hosts file secuessfully")
	}

	f, err := os.OpenFile("/etc/hosts", os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		log.Println("error while open hosts files")
	}
	p := "%s.%s.svc.cluster.local"
	_, err = f.WriteString("# -----write by vpn-----\n")
	if err != nil {
		log.Printf("write spliter failed. error: %v\n", err)
	}
	for _, svc := range list.Items {
		domain := fmt.Sprintf(p, svc.Name, svc.Namespace)
		ip := svc.Spec.ClusterIP
		_, err = f.WriteString(ip + " " + domain + "\n")
		if err != nil {
			log.Printf("write dns record failed. error: %v\n", err)
		}
	}
	_, err = f.WriteString("#--------end by vpn------\n")
	if err != nil {
		log.Printf("write spliter end failed. error: %v\n", err)
	}

	if err = f.Sync(); err != nil {
		log.Printf("sync hosts failed. error: %v\n", err)
	}

	if err = f.Close(); err != nil {
		log.Printf("close hosts failed. error: %v\n", err)
	}
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
