/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	"errors"
	"fmt"
	errors2 "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"net"
	"nocalhost/internal/nhctl/vpn/dns"
	"nocalhost/internal/nhctl/vpn/pkg/handler"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type ConnectOptions struct {
	Ctx              context.Context `json:"-"`
	Uid              string
	KubeconfigPath   string
	KubeconfigBytes  []byte
	Namespace        string
	Workloads        []string
	clientset        *kubernetes.Clientset
	restclient       *rest.RESTClient
	config           *rest.Config
	factory          cmdutil.Factory
	cidrs            []*net.IPNet
	localTunIP       *net.IPNet
	trafficManagerIP net.IP
	dhcp             *remote.DHCPManager
	log              *log.Logger
}

func (c *ConnectOptions) GetLogger() *log.Logger {
	if l := c.Ctx.Value("logger"); l != nil {
		return l.(*log.Logger)
	} else if c.log != nil {
		return c.log
	} else {
		c.log = util.NewLogger(os.Stdout)
		return c.log
	}
}

func (c *ConnectOptions) SetLogger(logger *log.Logger) {
	c.log = logger
}

func (c *ConnectOptions) IsSameUid(another *ConnectOptions) bool {
	return c.Uid == another.Uid
}

func (c *ConnectOptions) IsEmpty() bool {
	return c == nil || len(c.Uid) == 0
}

func (c *ConnectOptions) GetClientSet() *kubernetes.Clientset {
	return c.clientset
}

func (c *ConnectOptions) WaitTrafficManagerToAssignAnIP(log *log.Logger) error {
	podInterface := c.clientset.CoreV1().Pods(c.Namespace)
	if err := retry.OnError(wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1.0,
		Steps:    120,
		Cap:      120 * time.Second,
	}, func(err error) bool {
		return err != nil
	}, func() error {
		_, err := podInterface.Get(c.Ctx, util.TrafficManager, metav1.GetOptions{})
		return err
	}); err != nil {
		return err
	}
	w, err := podInterface.Watch(c.Ctx, metav1.SingleObject(metav1.ObjectMeta{Name: util.TrafficManager}))
	if err != nil {
		return err
	}
	defer w.Stop()
	var phase v1.PodPhase
	for {
		select {
		case e := <-w.ResultChan():
			if e.Type == watch.Deleted {
				return errors.New("traffic manager is deleted")
			}
			if podT, ok := e.Object.(*v1.Pod); ok {
				if phase != podT.Status.Phase {
					log.Infof("traffic manager is %s...", podT.Status.Phase)
				}
				if len(podT.Status.PodIP) != 0 {
					return nil
				}
				phase = podT.Status.Phase
			}
		case <-time.Tick(time.Minute * 5):
			return errors.New("wait for pod traffic manager to assign an ip timeout")
		}
	}
}

func (c *ConnectOptions) RentIP(random bool) (ip *net.IPNet, err error) {
	for i := 0; i < 5; i++ {
		if ip, err = c.dhcp.RentIP(random); err == nil {
			return
		}
	}
	return nil, errors.New("can not rent ip")
}

func (c *ConnectOptions) ReleaseIP() error {
	configMap, err := c.clientset.CoreV1().ConfigMaps(c.Namespace).Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	//split := strings.Split(get.Data["DHCP"], ",")
	used := remote.FromStringToDHCP(configMap.Data[util.DHCP])
	if rentips, found := used[util.GetMacAddress().String()]; found {
		if err = c.dhcp.ReleaseIP(rentips.List()...); err != nil {
			return err
		}
	}
	delete(used, util.GetMacAddress().String())
	configMap.Data[util.DHCP] = remote.ToString(used)
	_, err = c.clientset.CoreV1().ConfigMaps(c.Namespace).Update(context.Background(), configMap, metav1.UpdateOptions{})
	return err
}

func (c *ConnectOptions) createRemoteInboundPod() error {
	for _, workload := range c.Workloads {
		if len(workload) > 0 {
			shadowTunIP, err := c.RentIP(true)
			if err != nil {
				return err
			}
			err = CreateInboundPod(
				c.Ctx,
				c.factory,
				c.clientset,
				c.Namespace,
				workload,
				c.localTunIP.IP.String(),
				c.trafficManagerIP.String(),
				shadowTunIP.String(),
				util.RouterIP.String(),
			)
			if err != nil {
				c.GetLogger().Errorf("error while reversing resource: %s, error: %s", workload, err)
				return err
			}
		}
	}
	return nil
}

func (c *ConnectOptions) RemoveInboundPod() error {
	for _, workload := range c.Workloads {
		sc, err := getHandler(c.factory, c.clientset, c.Namespace, workload, nil)
		if err != nil {
			return fmt.Errorf(
				"error while get handler of resource: %s in namespace: %s, error: %v",
				workload, c.Namespace, err)
		}
		if err = sc.Rollback(false); err != nil {
			return fmt.Errorf(
				"error while reset reverse resource: %s in namespace: %s, error: %v",
				workload, c.Namespace, err)
		}
	}
	return nil
}

func getHandler(
	factory cmdutil.Factory,
	clientset *kubernetes.Clientset,
	namespace,
	workload string,
	config *handler.PodRouteConfig,
) (handler.Handler, error) {
	info, err := util.GetUnstructuredObject(factory, namespace, workload)
	if err != nil {
		return nil, err
	}
	gvk := info.Mapping.Resource
	svcType := fmt.Sprintf("%s.%s.%s", gvk.Resource, gvk.Version, gvk.Group)
	var sc handler.Handler
	switch svcType {
	case "services.v1.":
		sc = handler.NewServiceHandler(factory, clientset.CoreV1().Services(info.Namespace), info, config)
	case "pods.v1.":
		sc = handler.NewPodHandler(factory, clientset.CoreV1().Pods(info.Namespace), info, config)
	default:
		sc = handler.NewUnstructuredHandler(factory, info, config)
	}
	return sc, nil
}

func (c *ConnectOptions) InitDHCP(ctx context.Context) error {
	c.dhcp = remote.NewDHCPManager(c.clientset, c.Namespace, &util.RouterIP)
	cm, err := c.dhcp.InitDHCPIfNecessary(ctx)
	if err != nil {
		return err
	}
	c.Uid = string(cm.GetUID())
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
	return nil
}

func (c *ConnectOptions) DoConnect(ctx context.Context) (chan error, error) {
	var err error
	if err = util.WaitPortToBeFree(10800, time.Second*5); err != nil {
		return nil, err
	}
	c.trafficManagerIP, err = createOutboundRouterPodIfNecessary(c.clientset, c.Namespace, &util.RouterIP, c.cidrs, c.GetLogger())
	if err != nil {
		return nil, errors2.WithStack(err)
	}
	c.GetLogger().Info("your ip is " + c.localTunIP.IP.String())
	if err = c.portForward(ctx); err != nil {
		return nil, err
	}
	return c.startLocalTunServe(ctx)
}

func (c *ConnectOptions) DoReverse(ctx context.Context) error {
	pod, err := c.clientset.CoreV1().Pods(c.Namespace).Get(ctx, util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return errors.New("can not found router pod")
	}
	if len(pod.Status.PodIP) == 0 {
		return errors.New("can not found router ip while reverse resource")
	}
	c.trafficManagerIP = net.ParseIP(pod.Status.PodIP)
	return c.createRemoteInboundPod()
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

func (c *ConnectOptions) portForward(ctx context.Context) error {
	var readyChan = make(chan struct{}, 1)
	var errChan = make(chan error, 1)
	var first = true
	go func(ctx context.Context) {
		for ctx.Err() == nil {
			func() {
				defer func() {
					if err := recover(); err != nil {
						c.GetLogger().Warnf("port-forward recover error: %v, ignore", err)
					}
				}()
				if !first {
					readyChan = make(chan struct{}, 1)
				}
				first = false
				//stopChan := make(chan struct{}, 1)
				//remote.CancelFunctions = append(remote.CancelFunctions, func() {
				//	defer func() {
				//		if err := recover(); err != nil {
				//		}
				//	}()
				//	close(stopChan)
				//})
				err := util.PortForwardPod(
					c.config,
					c.restclient,
					util.TrafficManager,
					c.Namespace,
					"10800:10800",
					readyChan,
					ctx.Done(),
				)
				if apierrors.IsNotFound(err) {
					c.GetLogger().Errorf("can not found traffic manager, err: %v, exiting", err)
					return
				}
				if err != nil {
					if strings.Contains(err.Error(), "unable to listen on any of the requested ports") ||
						strings.Contains(err.Error(), "address already in use") {
						errChan <- err
						runtime.Goexit()
					}
					c.GetLogger().Errorf("port-forward occurs error, err: %v, retrying", err)
					time.Sleep(time.Second * 2)
				}
			}()
		}
	}(ctx)
	c.GetLogger().Infoln("port-forwarding...")
	select {
	case <-readyChan:
		c.GetLogger().Infoln("port forward 10800:10800 ready")
		return nil
	case err := <-errChan:
		c.GetLogger().Errorf("port-forward error, err: %v", err)
		return err
	case <-time.Tick(time.Second * 30):
		return errors.New("wait port forward 10800:10800 to be ready timeout")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ConnectOptions) startLocalTunServe(ctx context.Context) (chan error, error) {
	if util.IsWindows() {
		c.localTunIP.Mask = net.CIDRMask(0, 32)
	} else {
		c.localTunIP.Mask = net.CIDRMask(24, 32)
	}
	var list = []string{util.RouterIP.String()}
	for _, cidr := range c.cidrs {
		list = append(list, cidr.String())
	}
	route := Route{
		ServeNodes: []string{
			fmt.Sprintf("tun://:8421/127.0.0.1:8421?net=%s&route=%s",
				c.localTunIP.String(), strings.Join(list, ",")),
		},
		ChainNode: "tcp://127.0.0.1:10800",
		Retries:   5,
	}
	errChan, err := Start(ctx, route)
	if err != nil {
		return nil, errors2.WithStack(err)
	}
	select {
	case err = <-errChan:
		if err != nil {
			return nil, errors2.WithStack(err)
		}
	default:
	}
	c.GetLogger().Infof("tunnel create successfully")
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
		return nil, errors2.WithStack(err)
	}
	//c.detectConflictDevice()
	log.Info("setup DNS service successfully")
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
	remote.CancelFunctions = append(remote.CancelFunctions, func() {
		select {
		case c <- errors.New("cancelled"):
		default:
		}
	})
	for i := range routers {
		go func(ctx context.Context, i int, c chan error) {
			if err = routers[i].Serve(ctx); err != nil {
				log.Debugln(err)
				select {
				case c <- err:
				default:
				}
			}
		}(ctx, i, c)
	}

	return c, nil
}

func getCIDR(clientset *kubernetes.Clientset, namespace string) ([]*net.IPNet, error) {
	var CIDRList []*net.IPNet
	// get pod CIDR from node spec
	if nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{}); err == nil {
		var podCIDRs = sets.NewString()
		for _, node := range nodeList.Items {
			if node.Spec.PodCIDRs != nil {
				podCIDRs.Insert(node.Spec.PodCIDRs...)
			}
			if len(node.Spec.PodCIDR) != 0 {
				podCIDRs.Insert(node.Spec.PodCIDR)
			}
		}
		for _, podCIDR := range podCIDRs.List() {
			if _, CIDR, err := net.ParseCIDR(podCIDR); err == nil {
				CIDRList = append(CIDRList, CIDR)
			}
		}
	}
	// if node spec can not contain pod IP
	if podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, pod := range podList.Items {
			if pod.Spec.HostNetwork {
				continue
			}
			if ip := net.ParseIP(pod.Status.PodIP); ip != nil {
				var contains bool
				for _, CIDR := range CIDRList {
					if CIDR.Contains(ip) {
						contains = true
						break
					}
				}
				if !contains {
					mask := net.CIDRMask(24, 32)
					CIDRList = append(CIDRList, &net.IPNet{IP: ip.Mask(mask), Mask: mask})
				}
			}
		}
	}
	// pod CIDR maybe is not same with service CIDR
	if serviceList, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, service := range serviceList.Items {
			if ip := net.ParseIP(service.Spec.ClusterIP); ip != nil {
				var contains bool
				for _, CIDR := range CIDRList {
					if CIDR.Contains(ip) {
						contains = true
						break
					}
				}
				if !contains {
					mask := net.CIDRMask(16, 32)
					CIDRList = append(CIDRList, &net.IPNet{IP: ip.Mask(mask), Mask: mask})
				}
			}
		}
	}

	// remove duplicate CIDR
	result := make([]*net.IPNet, 0)
	set := sets.NewString()
	for _, cidr := range CIDRList {
		if !set.Has(cidr.String()) {
			set.Insert(cidr.String())
			result = append(result, cidr)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("can not found any CIDR")
	}
	return result, nil
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
	c.KubeconfigBytes, _ = ioutil.ReadFile(c.KubeconfigPath)
	return
}

func (c *ConnectOptions) reset() error {
	return nil
}

// GenerateTunIP TODO optimize code, can use patch ?
func (c *ConnectOptions) GenerateTunIP(ctx context.Context) error {
	defer func() {
		if c.localTunIP != nil {
			if util.IsWindows() {
				c.localTunIP.Mask = net.CIDRMask(0, 32)
			} else {
				c.localTunIP.Mask = net.CIDRMask(24, 32)
			}
		}
	}()
	get, err := c.clientset.CoreV1().ConfigMaps(c.Namespace).Get(ctx, util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	mac2IP := remote.FromStringToMac2IP(get.Data[util.MacToIP])
	if ip := mac2IP.GetIPByMac(util.GetMacAddress().String()); len(ip) != 0 {
		c.localTunIP = &net.IPNet{IP: net.ParseIP(ip), Mask: net.CIDRMask(24, 32)}
		return nil
	}
	c.localTunIP, err = c.dhcp.RentIP(false)
	if err != nil {
		return err
	}
	get, err = c.clientset.CoreV1().ConfigMaps(c.Namespace).Get(context.TODO(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	data := mac2IP.AddMacToIPRecord(util.GetMacAddress().String(), c.localTunIP.IP).ToString()
	_, err = c.clientset.CoreV1().ConfigMaps(c.Namespace).Patch(
		context.TODO(),
		get.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf("{\"data\":{\"%s\":\"%s\"}}", util.MacToIP, data)),
		metav1.PatchOptions{},
	)
	return err
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
	h, err := getHandler(c.factory, c.clientset, c.Namespace, c.Workloads[0], nil)
	if err != nil {
		return false
	}
	pod, err := h.GetPod()
	if err != nil {
		return false
	}
	_, err = util.Shell(
		c.clientset,
		c.restclient,
		c.config,
		pod[0].Name,
		handler.VPN,
		c.Namespace,
		fmt.Sprintf("ping %s -c 4", c.localTunIP),
	)
	return err == nil
}

func (c *ConnectOptions) Shell(_ context.Context, workload string) (string, error) {
	if len(workload) == 0 {
		workload = c.Workloads[0]
	}
	h, err := getHandler(c.factory, c.clientset, c.Namespace, workload, nil)
	if err != nil {
		return "", err
	}
	pod, err := h.GetPod()
	if err != nil {
		return "", err
	}
	return util.Shell(
		c.clientset,
		c.restclient,
		c.config,
		pod[0].Name,
		handler.VPN,
		c.Namespace,
		"ping -c 4 "+c.localTunIP.IP.String(),
	)
}

func (c *ConnectOptions) detectConflictDevice() {
	tun := os.Getenv("tunName")
	if len(tun) == 0 {
		return
	}
	if err := DetectAndDisableConflictDevice(tun); err != nil {
		log.Warnf("error occours while disable conflict devices, err: %v", err)
	}
}

func (c *ConnectOptions) GetUnstructuredObject(workload string) (*resource.Info, error) {
	return util.GetUnstructuredObject(c.factory, c.Namespace, workload)
}
