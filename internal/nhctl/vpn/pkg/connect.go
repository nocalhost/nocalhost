package pkg

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"net"
	"nocalhost/internal/nhctl/vpn/dns"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type ConnectOptions struct {
	KubeconfigPath string
	Namespace      string
	Workloads      []string
	nodeConfig     Route
	clientset      *kubernetes.Clientset
	restclient     *rest.RESTClient
	config         *rest.Config
	factory        cmdutil.Factory
	cidrs          []*net.IPNet
	routerIP       string
	dhcp           *remote.DHCPManager
	ipUsed         []*net.IPNet
}

func (c *ConnectOptions) GetClientSet() *kubernetes.Clientset {
	return c.clientset
}

func (c *ConnectOptions) RentIP(random bool) (ip *net.IPNet, err error) {
	if random {
		ip, err = c.dhcp.RentIPRandom()
	}
	ip, err = c.dhcp.RentIPBaseNICAddress()
	if err != nil {
		return
	}
	c.ipUsed = append(c.ipUsed, ip)
	return
}

func (c *ConnectOptions) ReleaseIP() error {
	var err error
	for _, ip := range c.ipUsed {
		if err = c.dhcp.ReleaseIpToDHCP(ip); err != nil {
			return err
		}
	}
	return nil
}

func (c *ConnectOptions) createRemoteInboundPod() error {
	var list []string
	for _, ipNet := range c.cidrs {
		list = append(list, ipNet.String())
	}

	tunIp, err := c.RentIP(false)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	for _, workload := range c.Workloads {
		if len(workload) > 0 {
			wg.Add(1)
			go func(finalWorkload string) {
				defer wg.Done()
				lock.Lock()
				virtualShadowIp, _ := c.RentIP(true)
				lock.Unlock()

				err = CreateInboundPod(
					c.factory,
					c.clientset,
					c.Namespace,
					finalWorkload,
					tunIp.IP.String(),
					c.routerIP,
					virtualShadowIp.String(),
					util.RouterIP.String(),
				)

				if err != nil {
					log.Error(err)
				}
			}(workload)
		}
	}
	wg.Wait()
	if util.IsWindows() {
		tunIp.Mask = net.CIDRMask(0, 32)
	} else {
		tunIp.Mask = net.CIDRMask(24, 32)
	}

	list = append(list, util.RouterIP.String())

	c.nodeConfig.ChainNode = "tcp://127.0.0.1:10800"
	c.nodeConfig.ServeNodes = []string{fmt.Sprintf("tun://:8421/127.0.0.1:8421?net=%s&route=%s", tunIp.String(), strings.Join(list, ","))}

	log.Info("your ip is " + tunIp.String())
	return nil
}

func (c *ConnectOptions) DoConnect(ctx context.Context) (chan error, error) {
	var err error
	c.cidrs, err = getCIDR(c.clientset, c.Namespace)
	if err != nil {
		return nil, err
	}
	c.routerIP, err = CreateOutboundRouterPodIfNecessary(c.clientset, c.Namespace, &util.RouterIP, c.cidrs)
	if err != nil {
		return nil, err
	}
	c.dhcp = remote.NewDHCPManager(c.clientset, c.Namespace, &util.RouterIP)
	if err = c.dhcp.InitDHCPIfNecessary(); err != nil {
		return nil, err
	}
	if err = c.createRemoteInboundPod(); err != nil {
		return nil, err
	}
	c.portForward(ctx)
	return c.startLocalTunServe(ctx)
}

func (c *ConnectOptions) heartbeats(ctx context.Context) {
	go func() {
		tick := time.Tick(time.Second * 15)
		c2 := make(chan struct{}, 1)
		c2 <- struct{}{}
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				c2 <- struct{}{}
			case <-c2:
				_ = exec.Command("ping", "-c", "4", util.IpRange.String()).Run()
			}
		}
	}()
}

func (c *ConnectOptions) portForward(ctx context.Context) {
	var readyChanRef *chan struct{}
	go func(ctx context.Context) {
		for ctx.Err() == nil {
			func() {
				defer func() {
					if err := recover(); err != nil {
						log.Warnf("recover error: %v, ignore", err)
					}
				}()
				readChan := make(chan struct{})
				stopChan := make(chan struct{})
				remote.CancelFunctions = append(remote.CancelFunctions, func() { stopChan <- struct{}{} })
				readyChanRef = &readChan
				err := util.PortForwardPod(
					c.config,
					c.restclient,
					util.TrafficManager,
					c.Namespace,
					"10800:10800",
					readChan,
					stopChan,
				)
				if apierrors.IsNotFound(err) {
					log.Errorln("can not found port-forward resource, err: %v, exiting", err)
					return
				}
				if err != nil {
					log.Errorf("port-forward occurs error, err: %v, retrying", err)
					time.Sleep(time.Second * 2)
				}
			}()
		}
	}(ctx)
	for readyChanRef == nil {
	}
	<-*readyChanRef
	log.Info("port forward ready")
}

func (c *ConnectOptions) startLocalTunServe(ctx context.Context) (chan error, error) {
	errChan, err := Start(ctx, c.nodeConfig)
	if err != nil {
		log.Fatal(err)
	}

	if util.IsWindows() {
		if !util.FindRule() {
			util.AddFirewallRule()
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.Tick(time.Second * 10):
					util.DeleteWindowsFirewallRule()
				}
			}
		}()
	}
	c.heartbeats(ctx)
	if err = c.setupDNS(); err != nil {
		return nil, err
	}
	log.Info("dns service ok")
	return errChan, nil
}

func (c ConnectOptions) setupDNS() error {
	relovConf, err := dns.GetDNSServiceIPFromPod(c.clientset, c.restclient, c.config, util.TrafficManager, c.Namespace)
	if err != nil {
		return err
	}
	if err = dns.SetupDNS(relovConf); err != nil {
		return err
	}
	return nil
}

func Start(ctx context.Context, r Route) (chan error, error) {
	routers, err := r.GenRouters()
	if err != nil {
		return nil, err
	}

	if len(routers) == 0 {
		return nil, errors.New("invalid config")
	}
	c := make(chan error, len(routers))
	remote.CancelFunctions = append(remote.CancelFunctions, func() { c <- errors.New("exit") })
	for i := range routers {
		go func(ctx context.Context, i int, c chan error) {
			if err = routers[i].Serve(ctx); err != nil {
				log.Warn(err)
				c <- err
			}
		}(ctx, i, c)
	}

	return c, nil
}

func getCIDR(clientset *kubernetes.Clientset, namespace string) ([]*net.IPNet, error) {
	var cidrs []*net.IPNet
	if nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, node := range nodeList.Items {
			if _, ip, _ := net.ParseCIDR(node.Spec.PodCIDR); ip != nil {
				ip.Mask = net.CIDRMask(16, 32)
				ip.IP = ip.IP.Mask(ip.Mask)
				cidrs = append(cidrs, ip)
			}
		}
	}
	if serviceList, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, service := range serviceList.Items {
			if ip := net.ParseIP(service.Spec.ClusterIP); ip != nil {
				mask := net.CIDRMask(16, 32)
				cidrs = append(cidrs, &net.IPNet{IP: ip.Mask(mask), Mask: mask})
			}
		}
	}
	if podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, pod := range podList.Items {
			if ip := net.ParseIP(pod.Status.PodIP); ip != nil {
				mask := net.CIDRMask(16, 32)
				cidrs = append(cidrs, &net.IPNet{IP: ip.Mask(mask), Mask: mask})
			}
		}
	}
	result := make([]*net.IPNet, 0)
	tempMap := make(map[string]*net.IPNet)
	for _, cidr := range cidrs {
		if _, found := tempMap[cidr.String()]; !found {
			tempMap[cidr.String()] = cidr
			result = append(result, cidr)
		}
	}
	if len(result) != 0 {
		return result, nil
	}
	return nil, fmt.Errorf("can not found cidr")
}

func (c *ConnectOptions) InitClient() {
	var err error
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	configFlags.KubeConfig = &c.KubeconfigPath
	c.factory = cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(configFlags))

	if c.config, err = c.factory.ToRESTConfig(); err != nil {
		log.Fatal(err)
	}
	if c.restclient, err = c.factory.RESTClient(); err != nil {
		log.Fatal(err)
	}
	if c.clientset, err = c.factory.KubernetesClientSet(); err != nil {
		log.Fatal(err)
	}
	if len(c.Namespace) == 0 {
		if c.Namespace, _, err = c.factory.ToRawKubeConfigLoader().Namespace(); err != nil {
			log.Fatal(err)
		}
	}
	log.Infof("kubeconfig path: %s, namespace: %s, serivces: %v", c.KubeconfigPath, c.Namespace, c.Workloads)
}

func (c *ConnectOptions) reset() error {
	return nil
}
