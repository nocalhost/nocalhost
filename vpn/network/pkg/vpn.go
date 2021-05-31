package pkg

import (
	"context"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/exec"
	"log"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/test/nhctlcli"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

/**
 * 1) create dns server
 * 2) port-forward dns server
 * 3) generate ssh key
 * 4) sshuttle
 */
var config *restclient.Config
var clientset *kubernetes.Clientset

const DNSPOD = "dnsserver"

// the reason why using ref-count are all start operation will using single one dns pod, save resource
const RefCountKey = "ref-count"

func Start(option Options) {
	clientgoutils.Must(initClient(&option))
	addCleanUpResource(option)
	privateKeyPath := createOrDumpSshPrivateKeyDnsPodConfigMap(option.Namespace)
	dnsServer := createDnsPod(option.Namespace)
	// random port
	localSshPort, _ := ports.GetAvailablePort()
	c := make(chan struct{})
	go portForwardService(option, localSshPort, c)
	<-c
	updateRefCount(option, 1)
	log.Println("port forward ready")
	if runtime.GOOS == "windows" || os.Getenv("debug") != "" {
		sock5Port, _ := ports.GetAvailablePort()
		go sshOutbound(privateKeyPath, sock5Port, localSshPort, c)
		<-c
		// socks5h means dns resolve should in remote pod, not local
		fmt.Printf(`please export http_proxy=socks5h://127.0.0.1:%d, and the you can access cluster ip or domain`+"\n", sock5Port)
	} else {
		go sshuttleOutbound(dnsServer, privateKeyPath, localSshPort, GetCidr(), c)
		<-c
		log.Println("expose remote to local successfully, you can access your cluster network on your local environment")
	}
	Inbound(option, privateKeyPath)
	select {}
}

func parseCidr(ip string, cidrList *[]string) {
	split := strings.Split(ip, ".")
	if len(split) == 4 {
		p := "%s.%s.0.0/16"
		*cidrList = append(*cidrList, fmt.Sprintf(p, split[0], split[1]))
	}
}

func GetCidr() []string {
	var cidrList []string
	if nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, node := range nodeList.Items {
			if node.Spec.PodCIDR != "" {
				cidrList = append(cidrList, node.Spec.PodCIDR)
			}
			if len(node.Spec.PodCIDRs) > 0 {
				cidrList = append(cidrList, node.Spec.PodCIDRs...)
			}
		}
	} else {
		log.Printf("failed to get node's cidr")
	}
	if services, err := clientset.CoreV1().Services(v1.NamespaceAll).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, service := range services.Items {
			parseCidr(service.Spec.ClusterIP, &cidrList)
			for _, clusterIP := range service.Spec.ClusterIPs {
				parseCidr(clusterIP, &cidrList)
			}
		}
	} else {
		log.Printf("failed to get service's cidr")
	}

	if list, err := clientset.CoreV1().Endpoints(v1.NamespaceAll).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, endpoints := range list.Items {
			for _, subset := range endpoints.Subsets {
				for _, address := range subset.Addresses {
					parseCidr(address.IP, &cidrList)
				}
			}
		}
	} else {
		log.Println("failed to get endpoint's cidr")
	}

	return distinct(cidrList)
}

func distinct(strings []string) (result []string) {
	m := make(map[string]string)
	for _, s := range strings {
		m[s] = s
	}
	for _, s := range m {
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		result = append(result, "0/0")
	}
	return
}

/**
 * 1) scale deployment replicas to zero
 * 2) relabel new shadow pod, make sure traffic which from service will receive by shadow pod
 * 3) using another images to listen local and transfer traffic
 */
func Inbound(options Options, privateKeyPath string) {
	if options.ServiceName == "" || options.Namespace == "" || options.PortPairs == "" {
		log.Println("no need to expose local service to remote")
		return
	}
	log.Println("prepare to expose local service to remote")
	scaleDeploymentReplicasTo(options, 0)
	podShadow := createPodShadow(options)
	if podShadow == nil {
		log.Println("fail to create shadow")
		return
	}
	log.Printf("wait for shadow: %s to be ready...\n", podShadow.Name)
	WaitPodToBeStatus(options.Namespace, "name="+podShadow.Name, func(pod *v1.Pod) bool { return v1.PodRunning == pod.Status.Phase })

	localSshPort, err := ports.GetAvailablePort()
	if err != nil {
		log.Fatal(err)
	}
	readyChan := make(chan struct{})
	stopsChan := make(chan struct{})
	go func() {
		if err = portForwardPod(podShadow.Name, podShadow.Namespace, localSshPort, readyChan, stopsChan); err != nil {
			log.Printf("port forward error, exiting")
			panic(err)
		}
	}()
	<-readyChan
	log.Println("port forward ready")
	remote2Local := strings.Split(options.PortPairs, ",")
	wg := sync.WaitGroup{}
	for _, pair := range remote2Local {
		p := strings.Split(pair, ":")
		wg.Add(1)
		go sshReverseProxy(p[0], p[1], localSshPort, privateKeyPath, &wg)
	}
	wg.Wait()
	log.Println("expose local to remote successfully, you can develop now, if you not need it anymore, can using ctrl+c to stop it")
	// hang up
	select {}
}

func createPodShadow(options Options) *v1.Pod {
	service, err := clientset.CoreV1().Services(options.Namespace).Get(context.TODO(), options.ServiceName, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
	}
	labels := service.Spec.Selector
	// todo version
	newName := options.ServiceName + "-" + "shadow"
	deployment, err := clientset.AppsV1().Deployments(options.Namespace).Get(context.TODO(), options.ServiceName, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return nil
	}
	forkSshConfigMapToNamespace(options)
	pod := newPod(newName, options.Namespace, labels, deployment.Spec.Template.Spec.Containers[0].Ports)
	cleanShadow(options, true)
	pods, err := clientset.CoreV1().Pods(options.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		log.Println(err)
	}
	return pods
}

func forkSshConfigMapToNamespace(options Options) {
	_, err := clientset.CoreV1().ConfigMaps(options.Namespace).Get(context.TODO(), DNSPOD, metav1.GetOptions{})
	if err == nil {
		log.Printf("ssh configmap already exist in namespace: %s, no need to fork it\n", options.Namespace)
		return
	}
	configMap, err := clientset.CoreV1().ConfigMaps(options.Namespace).Get(context.TODO(), DNSPOD, metav1.GetOptions{})
	if err != nil {
		log.Printf("can't find configmap: %s in namespace: %s\n", DNSPOD, options.Namespace)
		return
	}
	newConfigmap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DNSPOD,
			Namespace: options.Namespace,
		},
		Data: configMap.Data,
	}
	_, err = clientset.CoreV1().ConfigMaps(options.Namespace).Create(context.TODO(), &newConfigmap, metav1.CreateOptions{})
	if err == nil {
		log.Printf("fork configmap secuessfully")
	} else {
		log.Printf("fork configmap failed")
	}
}

var stopChan = make(chan os.Signal)

func addCleanUpResource(options Options) {
	signal.Notify(stopChan, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL /*, syscall.SIGSTOP*/)
	go func() {
		<-stopChan
		log.Println("prepare to exit, cleaning up")
		cleanUp(Option)
		scaleDeploymentReplicasTo(options, 1)
		cleanShadow(options, false)
		cleanSsh()
		log.Println("clean up successful")
		os.Exit(0)
	}()
}

func cleanSsh() {
	var err error
	if runtime.GOOS == "windows" {
		err = exec.New().Command("taskkill.exe", "/f", "/im", "Ssh.exe").Run()
	} else {
		cmd := "ps -ef | grep ssh | grep -v grep | awk -F ' ' '{print$2}' | xargs kill"
		err = exec.New().Command("/bin/bash", "-c", cmd).Run()
	}
	if err != nil {
		log.Println(err)
	}
}

func cleanShadow(options Options, wait bool) {
	shadowName := options.ServiceName + "-" + "shadow"
	err := clientset.CoreV1().Pods(options.Namespace).Delete(context.TODO(), shadowName, metav1.DeleteOptions{})
	if !wait {
		return
	}
	log.Printf("waiting for pod: %s to be deleted...\n", shadowName)
	if err == nil {
		w, err := clientset.CoreV1().Pods(options.Namespace).Watch(context.TODO(), metav1.ListOptions{
			LabelSelector: "name=" + shadowName,
			Watch:         true,
		})
		if err != nil {
			log.Println(err)
		}
	out:
		for {
			select {
			case event := <-w.ResultChan():
				if watch.Deleted == event.Type {
					break out
				}
			}
		}
		log.Printf("delete pod: %s suecessfully\n", shadowName)
	} else {
		log.Println("not found shadow pod, no need to delete it")
	}
}

// vendor/k8s.io/kubectl/pkg/polymorphichelpers/rollback.go:99
func updateRefCount(option Options, increment int) {
	get, err := clientset.CoreV1().
		ConfigMaps(option.Namespace).
		Get(context.TODO(), DNSPOD, metav1.GetOptions{})
	if err != nil {
		log.Println("this should not happened")
		return
	}
	curCount, err := strconv.Atoi(get.GetAnnotations()[RefCountKey])
	if err != nil {
		curCount = 0
	}

	patch, _ := json.Marshal([]interface{}{
		map[string]interface{}{
			"op":    "replace",
			"path":  "/metadata/annotations/" + RefCountKey,
			"value": strconv.Itoa(curCount + increment),
		},
	})
	_, err = clientset.CoreV1().ConfigMaps(option.Namespace).Patch(context.TODO(),
		DNSPOD, types.JSONPatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Printf("update ref count error, error: %v\n", err)
	} else {
		log.Println("update ref count successfully")
	}
}

func PreCheck() error {
	if _, err := osexec.LookPath("kubectl"); err != nil {
		log.Println("can not found kubectl, please install it previously")
		return err
	}
	if _, err := osexec.LookPath("ssh"); err != nil {
		log.Println("can not found ssh, please install it previously")
		return err
	}
	if _, err := osexec.LookPath("sshuttle"); err != nil {
		if runtime.GOOS == "macos" {
			if _, err = osexec.LookPath("brew"); err == nil {
				_ = os.Setenv("HOMEBREW_NO_AUTO_UPDATE", "true")
				log.Println("try to using homebrew to install sshuttle")
				cmd := osexec.Command("brew", "install", "sshuttle")
				_, stderr, err2 := nhctlcli.Runner.RunWithRollingOut(cmd)
				if err2 != nil {
					log.Printf("try to install sshuttle failed, error: %v, stderr: %s", err2, stderr)
					return nil
				} else {
					fmt.Println("install sshuttle successfully")
				}
			}
		}
	}
	return nil
}

func cleanUp(option Options) {
	updateRefCount(option, -1)
	configMap, err := clientset.CoreV1().
		ConfigMaps(option.Namespace).
		Get(context.TODO(), DNSPOD, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return
	}
	refCount, err := strconv.Atoi(configMap.GetAnnotations()[RefCountKey])
	if err != nil {
		log.Println(err)
	}
	// if refcount is less than zero or equals to zero, means no body will using this dns pod, so clean it
	if refCount <= 0 {
		log.Println("refCount is zero, prepare to clean up resource")
		_ = clientset.CoreV1().ConfigMaps(option.Namespace).Delete(context.TODO(), DNSPOD, metav1.DeleteOptions{})
		_ = clientset.AppsV1().Deployments(option.Namespace).Delete(context.TODO(), DNSPOD, metav1.DeleteOptions{})
		_ = clientset.CoreV1().Services(option.Namespace).Delete(context.TODO(), DNSPOD, metav1.DeleteOptions{})
	}
}

func initClient(options *Options) error {
	if _, err := os.Stat(options.Kubeconfig); err != nil {
		log.Println("using default kubeconfig")
		options.Kubeconfig = filepath.Join(HomeDir(), ".kube", "config")
	}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: options.Kubeconfig}, &clientcmd.ConfigOverrides{})
	if options.Namespace == "" {
		namespace, _, _ := clientConfig.Namespace()
		options.Namespace = namespace
	}
	var err error
	config, err = clientConfig.ClientConfig()
	if err != nil {
		return err
	}
	clientset, err = kubernetes.NewForConfig(config)
	return err
}

func CreateDNSPodDeployment(namespace string) {
	_, err := clientset.AppsV1().
		Deployments(namespace).
		Get(context.Background(), DNSPOD, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Println("prepare to create deployment: " + DNSPOD)
			_, err = clientset.AppsV1().
				Deployments(namespace).
				Create(context.Background(), newDnsPodDeployment(namespace), metav1.CreateOptions{})
			if err != nil {
				log.Println(err)
			}
			log.Println("create deployment successfully")
		} else {
			log.Fatal(err)
		}
	} else {
		log.Println("deployment already exist")
	}
	log.Println("wait for pod ready...")
	WaitPodToBeStatus(namespace, "app="+DNSPOD, func(pod *v1.Pod) bool {
		return v1.PodRunning == pod.Status.Phase
	})
	log.Printf("pod: %s ready\n", DNSPOD)
}
func createOrDumpSshPrivateKeyDnsPodConfigMap(namespace string) string {
	file, _ := ioutil.TempFile("", "")
	privateKeyPath := file.Name()

	configmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), DNSPOD, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Println("try to generate ssh info")
			info, err := generateSshKey(privateKeyPath)
			if info == nil {
				panic(err)
			}
			log.Println("prepare to create configmap")
			sshConfigmap := newSshConfigmap(namespace, info.PrivateKeyBytes, info.PublicKeyBytes)
			_, err = clientset.CoreV1().ConfigMaps(namespace).
				Create(context.Background(), sshConfigmap, metav1.CreateOptions{})
			if err != nil {
				log.Println(err)
			}
			return privateKeyPath
		} else {
			log.Fatal(err)
			return ""
		}
	} else {
		log.Println("configmap already exist, dump private key")
		s := configmap.Data["privateKey"]
		err = ioutil.WriteFile(privateKeyPath, []byte(s), 0700)
		if err != nil {
			log.Println(err)
		}
		log.Println("dump private key ok")
		return privateKeyPath
	}
}

func createDnsPod(namespace string) (serviceIp string) {
	CreateDNSPodDeployment(namespace)
	service, err := clientset.CoreV1().Services(namespace).Get(context.Background(), DNSPOD, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Println("prepare to create service: " + DNSPOD)
			service, err = clientset.CoreV1().Services(namespace).
				Create(context.Background(), newDnsPodService(namespace), metav1.CreateOptions{})
		}
	} else {
		log.Println("service already exist")
		log.Println("service ip: " + service.Spec.ClusterIP)
	}
	serviceIp = service.Spec.ClusterIP
	return
}
