package network

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	clientgowatch "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/utils/exec"
	"log"
	"net/http"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/test/nhctlcli"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Sshuttle interface {
	Outbound()
	Inbound()
}

type vpn func(string)

func (s vpn) Outbound() {
	str := "sshuttle -r root@106.55.60.68 -e \"ssh -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -i /Users/naison/.ssh/mytke_id_rsa\" 192.168.0.0/26 -x 106.55.60.68"
	cmd := exec.New().Command(str)
	cmd.Start()
}

func (s vpn) Inbound() {
	_ = "kubectl expose deployment/nginx-deployment --type=NodePort --name=nginx-test"
	_ = "ssh -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -i /Users/naison/.ssh/mytke_id_rsa -R 31586:127.0.0.1:8888 root@106.55.60.68 -p 22"
}

/**
 * 1) create dns server
 * 2) port-forward dns server
 * 3) generate ssh key
 * 4) sshuttle
 */
var config *restclient.Config
var clientset *kubernetes.Clientset

const namespace = "default"
const name = "dnsserver"

// the reason why using ref-count are all start operation will using single one dns pod, save resource
const refCountKey = "ref-count"

var kubeconfig = flag.String("kubeconfig", "", "your k8s cluster kubeconfig path")

func Start() {
	flag.Parse()
	log.Printf("kubeconfig path: %s\n", *kubeconfig)
	err := preCheck()
	if err != nil {
		panic(err)
	}
	privateKeyPath := filepath.Join(HomeDir(), ".nh", "ssh", "private", "key")
	public := filepath.Join(HomeDir(), ".nh", "ssh", "private", "pub")
	sshInfo := generateSSH(privateKeyPath, public)
	initClient(*kubeconfig)
	addCleanUpResource()
	serviceIp, _ := createDnsPod(sshInfo)
	// random port
	port, err := ports.GetAvailablePort()
	if err != nil {
		log.Fatal(err)
	}
	go portForwardService(*kubeconfig, port)
	time.Sleep(3 * time.Second)
	updateRefCount(1)
	log.Println("port forward ready")
	sshuttle(serviceIp, privateKeyPath, port)
}

func addCleanUpResource() {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	go func() {
		<-ch
		log.Println("prepare to exit, cleaning up")
		cleanUp()
		log.Println("clean up successful")
		os.Exit(0)
	}()
}

// vendor/k8s.io/kubectl/pkg/polymorphichelpers/rollback.go:99
func updateRefCount(increment int) {
	get, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		log.Println("this should not happened")
		return
	}
	curCount, err := strconv.Atoi(get.GetAnnotations()[refCountKey])
	if err != nil {
		curCount = 0
	}

	patch, _ := json.Marshal([]interface{}{
		map[string]interface{}{
			"op":    "replace",
			"path":  "/metadata/annotations/" + refCountKey,
			"value": strconv.Itoa(curCount + increment),
		},
	})
	_, err = clientset.CoreV1().ConfigMaps(namespace).Patch(context.TODO(),
		name, types.JSONPatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Printf("update ref count error, error: %v\n", err)
	} else {
		log.Println("update ref count successfully")
	}
}

func preCheck() error {
	_, err := osexec.LookPath("kubectl")
	if err != nil {
		log.Println("can not found kubectl, please install it previously")
		return err
	}
	_, err = osexec.LookPath("sshuttle")
	if err != nil {
		if _, err = osexec.LookPath("brew"); err == nil {
			log.Println("try to install sshuttle")
			cmd := osexec.Command("brew", "install", "sshuttle")
			_, stderr, err2 := nhctlcli.Runner.RunWithRollingOut(cmd)
			if err2 != nil {
				log.Printf("try to install sshuttle failed, error: %v, stderr: %s", err2, stderr)
				return err2
			}
			fmt.Println("install sshuttle successfully")
		}
	}
	return nil
}

func cleanUp() {
	updateRefCount(-1)
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return
	}
	refCount, err := strconv.Atoi(configMap.GetAnnotations()[refCountKey])
	if err != nil {
		log.Println(err)
	}
	// if refcount is less than zero or equals to zero, means no body will using this dns pod, so clean it
	if refCount <= 0 {
		log.Println("refCount is zero, prepare to clean up resource")
		_ = clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		_ = clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		_ = clientset.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	}
}

func initClient(kubeconfigPath string) {
	if _, err := os.Stat(kubeconfigPath); err != nil {
		kubeconfigPath = filepath.Join(HomeDir(), ".kube", "config")
	}
	config, _ = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	clientset, _ = kubernetes.NewForConfig(config)
}

func createDnsPod(info *SSHInfo) (serviceIp, podName string) {
	configmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Println("prepare to create configmap")
			_, err = clientset.CoreV1().
				ConfigMaps(namespace).
				Create(context.Background(), newSshConfigmap(info.PrivateKeyBytes, info.PublicKeyBytes), metav1.CreateOptions{})
			if err != nil {
				log.Println(err)
			}
		} else {
			log.Fatal(err)
		}
	} else {
		log.Println("configmap already exist, dump private key")
		s := configmap.Data["privateKey"]
		err = ioutil.WriteFile(filepath.Join(HomeDir(), ".nh", "ssh", "private", "key"), []byte(s), 0700)
		if err != nil {
			log.Println(err)
		}
		log.Println("dump private key ok")
	}

	_, err = clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Println("prepare to create deployment: " + name)
			_, err = clientset.AppsV1().Deployments(namespace).Create(context.Background(), newDnsPodDeployment(), metav1.CreateOptions{})
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
	WaitPodToBeStatus(namespace, "app="+name, func(pod *v1.Pod) bool { return v1.PodRunning == pod.Status.Phase })
	list, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=" + name})
	if err != nil {
		log.Fatal(err)
	}
	podName = list.Items[0].Name
	log.Println("pod ready")
	service, err := clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Println("prepare to create service: " + name)
			service, err = clientset.CoreV1().Services(namespace).Create(context.Background(), newDnsPodService(), metav1.CreateOptions{})
		}
	} else {
		log.Println("service already exist")
		log.Println("service ip: " + service.Spec.ClusterIP)
	}
	serviceIp = service.Spec.ClusterIP
	return
}

func portForward(resource string, name string, port int32, readyChan, stopChan chan struct{}) (*portforward.PortForwarder, error) {
	log.Printf("%s-%s\n", resource, name)
	url := clientset.AppsV1().
		RESTClient().
		Post().
		Resource(resource).
		Namespace(namespace).
		Name(name).
		SubResource("portforward").
		URL()
	transport, upgrader, err := spdy.RoundTripperFor(config)
	log.Println("url: " + url.String())
	if err != nil {
		log.Println(err)
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
	out := new(bytes.Buffer)
	ports := []string{fmt.Sprintf("%d:%d", port, 22)}
	pf, err := portforward.New(dialer, ports, stopChan, readyChan, ioutil.Discard, out)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return pf, nil
}

func WaitPodToBeStatus(namespace string, label string, checker func(*v1.Pod) bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	watchlist := cache.NewFilteredListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		namespace,
		func(options *metav1.ListOptions) { options.LabelSelector = label })

	preConditionFunc := func(store cache.Store) (bool, error) {
		if len(store.List()) == 0 {
			return false, nil
		}
		for _, p := range store.List() {
			if !checker(p.(*v1.Pod)) {
				return false, nil
			}
		}
		return true, nil
	}
	conditionFunc := func(e watch.Event) (bool, error) { return checker(e.Object.(*v1.Pod)), nil }
	event, err := clientgowatch.UntilWithSync(ctx, watchlist, &v1.Pod{}, preConditionFunc, conditionFunc)
	if err != nil {
		log.Printf("wait pod has the label: %s to ready failed, error: %v, event: %v", label, err, event)
		return false
	}
	return true
}

func portForwardService(kubeconfigPath string, localPort int) {
	cmd := exec.New().
		CommandContext(
			context.TODO(),
			"kubectl",
			"port-forward",
			"service/dnsserver",
			strconv.Itoa(localPort)+":22",
			"--namespace",
			"default",
			"--kubeconfig", kubeconfigPath)
	cmd.Start()
	err := cmd.Wait()
	if err != nil {
		log.Println(err)
	}
}

func sshuttle(serviceIp, sshPrivateKeyPath string, port int) {
	args := []string{
		"-r",
		"root@127.0.0.1:" + strconv.Itoa(port),
		"-e", "ssh -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -i " + sshPrivateKeyPath,
		"0/0",
		"-x",
		"127.0.0.1",
		"--dns",
		"--to-ns",
		serviceIp,
	}
	cmd := osexec.CommandContext(context.TODO(), "sshuttle", args...)
	out, s, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	if err != nil {
		log.Printf("error: %v, stdout: %s, stderr: %s", err, out, s)
	} else {
		log.Printf(out)
	}
}

func newSshConfigmap(privateKey, publicKey []byte) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{refCountKey: "0"},
		},
		Data: map[string]string{"authorized": string(publicKey), "privateKey": string(privateKey)},
	}
}

func newDnsPodDeployment() *appsv1.Deployment {
	one := int32(1)
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "apps",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels:    map[string]string{"app": name},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  name,
						Image: "naison/dnsserver:latest",
						Ports: []v1.ContainerPort{
							{ContainerPort: 53, Protocol: v1.ProtocolTCP},
							{ContainerPort: 53, Protocol: v1.ProtocolUDP},
							{ContainerPort: 22},
						},
						ImagePullPolicy: v1.PullAlways,
						VolumeMounts: []v1.VolumeMount{{
							Name:      "ssh-key",
							MountPath: "/root",
						}},
					}},
					Volumes: []v1.Volume{{
						Name: "ssh-key",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: name,
								},
								Items: []v1.KeyToPath{{
									Key:  "authorized",
									Path: "authorized_keys",
								}},
							},
						},
					}},
				},
			},
		},
	}
}

func newDnsPodService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "tcp", Protocol: v1.ProtocolTCP, Port: 53, TargetPort: intstr.FromInt(53)},
				{Name: "udp", Protocol: v1.ProtocolUDP, Port: 53, TargetPort: intstr.FromInt(53)},
				{Name: "ssh", Port: 22, TargetPort: intstr.FromInt(22)},
			},
			Selector: map[string]string{"app": name},
			Type:     v1.ServiceTypeClusterIP,
		},
	}
}
