package pkg

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/util/podutils"
	"net"
	"nocalhost/internal/nhctl/vpn/dns"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"os/exec"
	"regexp"
	"sort"
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
	tunIP          *net.IPNet
	routerIP       string
	dhcp           *remote.DHCPManager
	ipUsed         []*net.IPNet
	Logger         *log.Logger
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

func (c *ConnectOptions) createRemoteInboundPod(tunIp *net.IPNet) error {
	var list []string
	for _, ipNet := range c.cidrs {
		list = append(list, ipNet.String())
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

				err := CreateInboundPod(
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
	return nil
}

func (c *ConnectOptions) RemoveInboundPod() error {
	tuple, parsed, err2 := util.SplitResourceTypeName(c.Workloads[0])
	if !parsed || err2 != nil {
		return errors.New("not need")
	}
	newName := ToInboundPodName(tuple.Resource, tuple.Name)
	var sc Scalable
	switch strings.ToLower(tuple.Resource) {
	case "deployment", "deployments":
		sc = NewDeploymentController(c.factory, c.clientset, c.Namespace, tuple.Name)
	case "statefulset", "statefulsets":
		sc = NewStatefulsetController(c.factory, c.clientset, c.Namespace, tuple.Name)
	case "replicaset", "replicasets":
		sc = NewReplicasController(c.factory, c.clientset, c.Namespace, tuple.Name)
	case "service", "services":
		sc = NewServiceController(c.factory, c.clientset, c.Namespace, tuple.Name)
	case "pod", "pods":
		sc = NewPodController(c.factory, c.clientset, c.Namespace, "pods", tuple.Name)
	default:
		sc = NewCustomResourceDefinitionController(c.factory, c.clientset, c.Namespace, tuple.Resource, tuple.Name)
	}
	if err := sc.Cancel(); err != nil {
		log.Warnln(err)
		return err
	}
	util.DeletePod(c.clientset, c.Namespace, newName)
	return nil
}

func (c *ConnectOptions) InitDHCP(ctx context.Context) error {
	c.dhcp = remote.NewDHCPManager(c.clientset, c.Namespace, &util.RouterIP)
	err := c.dhcp.InitDHCPIfNecessary(ctx)
	if err != nil {
		return err
	}
	return c.GenerateTunIP(ctx)
}

func (c *ConnectOptions) Prepare(ctx context.Context) error {
	var err error
	c.cidrs, err = getCIDR(c.clientset, c.Namespace)
	if err != nil {
		util.GetLoggerFromContext(ctx).Warnln(err)
		return err
	}
	if err = c.InitDHCP(ctx); err != nil {
		return err
	}
	var list []string
	for _, cidr := range c.cidrs {
		list = append(list, cidr.String())
	}
	list = append(list, util.RouterIP.String())

	c.nodeConfig.ChainNode = "tcp://127.0.0.1:10800"
	c.nodeConfig.ServeNodes = []string{
		fmt.Sprintf("tun://:8421/127.0.0.1:8421?net=%s&route=%s", c.tunIP.String(), strings.Join(list, ",")),
	}
	return err
}

func (c *ConnectOptions) DoConnect(ctx context.Context) (chan error, error) {
	var err error
	c.routerIP, err = CreateOutboundRouterPodIfNecessary(c.clientset, c.Namespace, &util.RouterIP, c.cidrs, c.Logger)
	if err != nil {
		return nil, err
	}
	c.Logger.Info("your ip is " + c.tunIP.String())
	if !util.IsPortListening(10800) {
		c.portForward(ctx)
	}
	return c.startLocalTunServe(ctx)
}

func (c *ConnectOptions) DoReverse(ctx context.Context) error {
	firstPod, _, err := polymorphichelpers.GetFirstPod(c.clientset.CoreV1(),
		c.Namespace,
		fields.OneTermEqualSelector("app", util.TrafficManager).String(),
		time.Second*5,
		func(pods []*v1.Pod) sort.Interface {
			return sort.Reverse(podutils.ActivePods(pods))
		},
	)
	if err != nil || firstPod == nil {
		return errors.New("can not found router ip while reverse")
	}
	c.routerIP = firstPod.Status.PodIP
	if err = c.createRemoteInboundPod(c.tunIP); err != nil {
		return err
	}
	return nil
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
						c.Logger.Warnf("recover error: %v, ignore", err)
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
					c.Logger.Errorln("can not found port-forward resource, err: %v, exiting", err)
					return
				}
				if err != nil {
					c.Logger.Errorf("port-forward occurs error, err: %v, retrying", err)
					//time.Sleep(time.Second * 2)
				}
			}()
		}
	}(ctx)
	for readyChanRef == nil {
	}
	<-*readyChanRef
	c.Logger.Info("port forward ready")
}

func (c *ConnectOptions) startLocalTunServe(ctx context.Context) (chan error, error) {
	errChan, err := Start(ctx, c.nodeConfig)
	if err != nil {
		return nil, err
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

func (c *ConnectOptions) InitClient(ctx context.Context) (err error) {
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	configFlags.KubeConfig = &c.KubeconfigPath
	c.factory = cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(configFlags))

	if c.config, err = c.factory.ToRESTConfig(); err != nil {
		return
	}
	if c.restclient, err = c.factory.RESTClient(); err != nil {
		return
	}
	if c.clientset, err = c.factory.KubernetesClientSet(); err != nil {
		return
	}
	if len(c.Namespace) == 0 {
		if c.Namespace, _, err = c.factory.ToRawKubeConfigLoader().Namespace(); err != nil {
			return
		}
	}
	util.GetLoggerFromContext(ctx).Infof("kubeconfig path: %s, namespace: %s, serivces: %v", c.KubeconfigPath, c.Namespace, c.Workloads)
	return
}

func (c *ConnectOptions) reset() error {
	return nil
}

// GenerateTunIP TODO optimize code, can use patch ?
func (c *ConnectOptions) GenerateTunIP(ctx context.Context) error {
	defer func() {
		if c.tunIP != nil {
			if util.IsWindows() {
				c.tunIP.Mask = net.CIDRMask(0, 32)
			} else {
				c.tunIP.Mask = net.CIDRMask(24, 32)
			}
		}
	}()
	get, err := c.clientset.CoreV1().ConfigMaps(c.Namespace).Get(context.TODO(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	dhcp := remote.ToDHCP(get.Data[util.MacToIP])
	if ip := dhcp.GetIP(); len(ip) != 0 {
		c.tunIP = &net.IPNet{IP: net.ParseIP(ip), Mask: net.CIDRMask(24, 32)}
		return nil
	}
	c.tunIP, err = c.dhcp.RentIPBaseNICAddress()
	if err != nil {
		return err
	}
	get, err = c.clientset.CoreV1().ConfigMaps(c.Namespace).Get(context.TODO(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	get.Data[util.MacToIP] = dhcp.RentIP(c.tunIP.IP).ToString()
	if _, err = c.clientset.CoreV1().ConfigMaps(c.Namespace).Update(context.TODO(), get, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (c *ConnectOptions) ConnectPingRemote() bool {
	var cmd *exec.Cmd
	if util.IsWindows() {
		cmd = exec.Command("ping", util.IpRange.String())
	} else {
		cmd = exec.Command("ping", "-c", "4", util.IpRange.String())
	}
	compile, _ := regexp.Compile("icmp_seq=(.*?) ttl=(.*?) time=(.*?)")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return compile.MatchString(string(output))
}

func (c *ConnectOptions) ReverePingLocal() bool {
	tuple, _, _ := util.SplitResourceTypeName(c.Workloads[0])
	_, err := util.Shell(
		c.clientset,
		c.restclient,
		c.config,
		ToInboundPodName(tuple.Resource, tuple.Name),
		c.Namespace,
		fmt.Sprintf("ping %s -c 4", c.tunIP),
	)
	return err == nil
}
