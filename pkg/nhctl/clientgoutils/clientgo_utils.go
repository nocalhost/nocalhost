package clientgoutils

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	appsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"nocalhost/pkg/nhctl/log"
)

type ClientGoUtils struct {
	kubeConfigFilePath string
	restConfig         *restclient.Config
	ClientSet          *kubernetes.Clientset
	dynamicClient      dynamic.Interface //
	TimeOut            time.Duration
	ClientConfig       clientcmd.ClientConfig
	//RestClient         *restclient.RESTClient
}

type PortForwardAPodRequest struct {
	// Pod is the selected pod for this port forwarding
	Pod corev1.Pod
	// LocalPort is the local port that will be selected to expose the PodPort
	LocalPort int
	// PodPort is the target port for the pod
	PodPort int
	// Steams configures where to write or read input from
	Streams genericclioptions.IOStreams
	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
}

// if timeout is set to 0, default timeout 5 minutes is used
func NewClientGoUtils(kubeConfigPath string, timeout time.Duration) (*ClientGoUtils, error) {
	var (
		err error
	)

	if kubeConfigPath == "" { // use default config
		//kubeConfigPath = fmt.Sprintf("%s/.kube/config", getHomePath())
		kubeConfigPath = filepath.Join(getHomePath(), ".kube", "config")

	}

	if timeout <= 0 {
		timeout = time.Minute * 5
	}
	client := &ClientGoUtils{
		kubeConfigFilePath: kubeConfigPath,
		TimeOut:            timeout,
	}

	client.ClientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}})

	client.restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	if client.ClientSet, err = kubernetes.NewForConfig(client.restConfig); err != nil {
		return nil, err
	}

	if client.dynamicClient, err = dynamic.NewForConfig(client.restConfig); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *ClientGoUtils) GetDynamicClient() dynamic.Interface {

	var restConfig *restclient.Config
	restConfig, _ = clientcmd.BuildConfigFromFlags("", c.kubeConfigFilePath)
	dyn, _ := dynamic.NewForConfig(restConfig)
	return dyn
}

func (c *ClientGoUtils) getRestConfig() (*restclient.Config, error) {
	return clientcmd.BuildConfigFromFlags("", c.kubeConfigFilePath)
}

// todo check use something more accurate
func (c *ClientGoUtils) CheckIfNamespaceIsAccessible(ctx context.Context, namespace string) (bool, error) {
	if namespace == "" {
		namespace, _ = c.GetDefaultNamespace()
	}
	_, err := c.GetDeployments(ctx, namespace)
	if err != nil {
		fmt.Printf("err:%v\n", err)
		return false, errors.New(fmt.Sprintf("namespace \"%s\" is unaccessible", namespace))
	} else {
		return true, nil
	}
}

func (c *ClientGoUtils) GetDefaultNamespace() (string, error) {
	ns, _, err := c.ClientConfig.Namespace()
	return ns, err
}

func (c *ClientGoUtils) createUnstructuredResource(namespace string, rawObj runtime.RawExtension, wait bool) error {
	obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return &TypedError{ErrorType: InvalidYaml, Mes: err.Error()}
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		//return errors.Wrap(err, fmt.Sprintf("[Invalid Yaml] fail to parse resource obj"))
		return &TypedError{ErrorType: InvalidYaml, Mes: err.Error()}
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	gr, err := restmapper.GetAPIGroupResources(c.ClientSet.Discovery())
	if err != nil {
		return err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	var dri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace != "" {
			unstructuredObj.SetNamespace(namespace)
		} else if unstructuredObj.GetNamespace() == "" {
			unstructuredObj.SetNamespace("default")
		}
		dri = c.GetDynamicClient().Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
	} else {
		dri = c.GetDynamicClient().Resource(mapping.Resource)
	}

	//obj2, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
	//log.Debug("background to todo")
	obj2, err := dri.Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("fail to create %s", unstructuredObj.GetName()))
	}

	fmt.Printf("%s/%s created\n", obj2.GetKind(), obj2.GetName())

	if wait {
		err = c.WaitJobToBeReady(obj2.GetNamespace(), obj2.GetName())
		if err != nil {
			PrintlnErr("fail to wait", err)
			return err
		}
	}
	return nil
}

func (c *ClientGoUtils) Create(yamlPath string, namespace string, wait bool, validate bool) error {
	if yamlPath == "" {
		return errors.New("yaml path can not be empty")
	}
	if namespace == "" {
		namespace, _ = c.GetDefaultNamespace()
	}

	filebytes, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(filebytes), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
			//return errors.Wrap(err, fmt.Sprintf("[Invalid Yaml] fail to decode %s", yamlPath))
		}
		err = c.createUnstructuredResource(namespace, rawObj, wait)
		if err != nil {
			if validate {
				return err
			}
			te, ok := err.(*TypedError)
			if ok {
				log.Warnf("Invalid yaml: %s", te.Mes)
			} else {
				log.Warnf("Fail to install manifest : %s", err.Error())
			}

		}
	}
	return nil
}

func (c *ClientGoUtils) GetDiscoveryClient() (*discovery.DiscoveryClient, error) {
	config, err := c.getRestConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(config)
}

func (c *ClientGoUtils) Discovery() {
	discoveryClient, err := c.GetDiscoveryClient()
	if err != nil {
		fmt.Println("failed to get discovery client")
		return
	}

	apiGroups, resourceList, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		fmt.Println("failed to discover")
		return
	}
	fmt.Println("the following api groups found:")
	for _, apiGroup := range apiGroups {
		fmt.Printf("%s %s %s\n", apiGroup.Kind, apiGroup.APIVersion, apiGroup.Name)
	}

	fmt.Println("the following resources found:")
	for _, list := range resourceList {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			fmt.Println("fail to get gv")
		}
		for _, resource := range list.APIResources {
			fmt.Printf("name:%30s group:%15s version:%s\n", resource.Name, gv.Group, gv.Version)
		}
	}
}

func (c *ClientGoUtils) GetDeploymentClient(namespace string) appsV1.DeploymentInterface {
	return c.ClientSet.AppsV1().Deployments(namespace)
}

func (c *ClientGoUtils) GetPodClient(namespace string) coreV1.PodInterface {
	return c.ClientSet.CoreV1().Pods(namespace)
}

func (c *ClientGoUtils) GetDeployment(ctx context.Context, namespace string, name string) (*v1.Deployment, error) {
	return c.GetDeploymentClient(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *ClientGoUtils) CheckDeploymentReady(ctx context.Context, namespace string, name string) (bool, error) {
	deployment, err := c.GetDeployment(ctx, namespace, name)
	if err != nil {
		return false, err
	}
	for _, c := range deployment.Status.Conditions {
		if c.Type == v1.DeploymentAvailable && c.Status == "True" {
			return true, nil
		}
	}
	return false, nil
}

func (c *ClientGoUtils) GetDeployments(ctx context.Context, namespace string) ([]v1.Deployment, error) {
	deps, err := c.GetDeploymentClient(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deps.Items, nil
}

func (c *ClientGoUtils) UpdateDeployment(ctx context.Context, namespace string, deployment *v1.Deployment, opts metav1.UpdateOptions, wait bool) (*v1.Deployment, error) {
	dep, err := c.GetDeploymentClient(namespace).Update(ctx, deployment, opts)
	if err != nil {
		return nil, err
	}
	if wait {
		err = c.WaitDeploymentToBeReady(namespace, dep.Name, c.TimeOut)
	}
	return dep, err
}

func (c *ClientGoUtils) ListPodsOfDeployment(namespace string, deployName string) ([]corev1.Pod, error) {
	podClient := c.GetPodClient(namespace)

	podList, err := podClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("failed to get pods, err: %v\n", err)
		return nil, err
	}

	result := make([]corev1.Pod, 0)

OuterLoop:
	for _, pod := range podList.Items {
		if pod.OwnerReferences == nil {
			continue
		}
		for _, ref := range pod.OwnerReferences {
			if ref.Kind != "ReplicaSet" {
				continue
			}
			rss, _ := c.GetReplicaSetsControlledByDeployment(context.TODO(), namespace, deployName)
			if rss == nil {
				continue
			}
			for _, rs := range rss {
				if rs.Name == ref.Name {
					result = append(result, pod)
					continue OuterLoop
				}
			}
		}
	}
	return result, nil
}

func (c *ClientGoUtils) GetSortedReplicaSetsByDeployment(ctx context.Context, namespace string, deployment string) ([]*v1.ReplicaSet, error) {
	rss, err := c.GetReplicaSetsControlledByDeployment(ctx, namespace, deployment)
	if err != nil {
		return nil, err
	}
	if rss == nil || len(rss) < 1 {
		return nil, nil
	}
	keys := make([]int, 0)
	for rs := range rss {
		keys = append(keys, rs)
	}
	sort.Ints(keys)
	results := make([]*v1.ReplicaSet, 0)
	for _, key := range keys {
		results = append(results, rss[key])
	}
	return results, nil
}

func (c *ClientGoUtils) GetReplicaSetsControlledByDeployment(ctx context.Context, namespace string, deploymentName string) (map[int]*v1.ReplicaSet, error) {
	var rsList *v1.ReplicaSetList
	replicaSetsClient := c.ClientSet.AppsV1().ReplicaSets(namespace)
	rsList, err := replicaSetsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	rsMap := make(map[int]*v1.ReplicaSet)
	for _, item := range rsList.Items {
		if item.OwnerReferences == nil {
			continue
		}
		for _, owner := range item.OwnerReferences {
			if owner.Name == deploymentName && item.Annotations["deployment.kubernetes.io/revision"] != "" {
				if revision, err := strconv.Atoi(item.Annotations["deployment.kubernetes.io/revision"]); err == nil {
					rsMap[revision] = item.DeepCopy()
				}
			}
		}
	}
	return rsMap, nil
}

func waitForJob(obj runtime.Object, name string) (bool, error) {
	o, ok := obj.(*batchv1.Job)
	if !ok {
		return true, errors.Errorf("expected %s to be a *batch.Job, got %T", name, obj)
	}

	for _, c := range o.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == "True" {
			fmt.Println("Job Completed")
			return true, nil
		} else if c.Type == batchv1.JobFailed && c.Status == "True" {
			fmt.Println("Job Failed")
			return true, errors.Errorf("job failed: %s", c.Reason)
		}
	}
	fmt.Println("Job is running")

	return false, nil
}

// syncthing
func (c *ClientGoUtils) CreateSecret(ctx context.Context, namespace string, secret *corev1.Secret, options metav1.CreateOptions) (*corev1.Secret, error) {
	return c.ClientSet.CoreV1().Secrets(namespace).Create(ctx, secret, options)
}

func (c *ClientGoUtils) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	return c.ClientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *ClientGoUtils) DeleteSecret(ctx context.Context, namespace, name string) error {
	return c.ClientSet.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *ClientGoUtils) GetPodsFromDeployment(ctx context.Context, namespace, name string) (*corev1.PodList, error) {
	deployment, err := c.ClientSet.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("deployment not found")
	}
	set := labels.Set(deployment.Spec.Selector.MatchLabels)
	pods, err := c.ClientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()})
	if err != nil {
		log.Fatalf("can not found pod under deployment %s", name)
	}
	return pods, nil
}

func (c *ClientGoUtils) PortForwardAPod(req PortForwardAPodRequest) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		req.Pod.Namespace, req.Pod.Name)
	clientConfig, err := c.ClientConfig.ClientConfig()
	if err != nil {
		log.Fatalf("get go client config fail, please check you kubeconfig")
	}
	hostIP := strings.TrimLeft(clientConfig.Host, "https://")

	transport, upgrader, err := spdy.RoundTripperFor(clientConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)}, req.StopCh, req.ReadyCh, req.Streams.Out, req.Streams.ErrOut)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}

func (c *ClientGoUtils) GetNodesList() (*corev1.NodeList, error) {
	nodes, err := c.ClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return &corev1.NodeList{}, err
	}
	return nodes, nil
}

func (c *ClientGoUtils) GetService(name, namespace string) (*corev1.Service, error) {
	service, err := c.ClientSet.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return &corev1.Service{}, err
	}
	return service, nil
}

func (c *ClientGoUtils) CheckExistNameSpace(name string) error {
	_, err := c.ClientSet.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *ClientGoUtils) CreateNameSpace(name string) error {
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := c.ClientSet.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
