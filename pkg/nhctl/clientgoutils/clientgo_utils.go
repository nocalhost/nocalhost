package clientgoutils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	appsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	//clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"log"
	"strconv"
	"time"
)

type ClientGoUtils struct {
	kubeConfigFilePath string
	//restConfig *restclient.Config
	ClientSet     *kubernetes.Clientset
	dynamicClient dynamic.Interface //
	TimeOut       time.Duration
	ClientConfig  clientcmd.ClientConfig
	//RestClient         *restclient.RESTClient
}

// if timeout is set to 0, default timeout 5 minutes is used
func NewClientGoUtils(kubeConfigPath string, timeout time.Duration) (*ClientGoUtils, error) {
	var (
		err        error
		restConfig *restclient.Config
	)

	if kubeConfigPath == "" { // use kubectl default config
		kubeConfigPath = fmt.Sprintf("%s/.kube/config", getHomePath())
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

	if restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil {
		return nil, err
	}

	if client.ClientSet, err = kubernetes.NewForConfig(restConfig); err != nil {
		return nil, err
	}

	if client.dynamicClient, err = dynamic.NewForConfig(restConfig); err != nil {
		return nil, err
	}

	return client, nil
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
		return false, errors.New(fmt.Sprintf("namespace \"%s\" is unaccessible", namespace))
	} else {
		return true, nil
	}
}

func (c *ClientGoUtils) GetDefaultNamespace() (string, error) {
	ns, _, err := c.ClientConfig.Namespace()
	return ns, err
}

func (c *ClientGoUtils) Delete(yamlPath string, namespace string) error {
	if yamlPath == "" {
		return errors.New("yaml path can not be empty")
	}
	if namespace == "" {
		namespace = "default"
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
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			log.Print(err)
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(c.ClientSet.Discovery())
		if err != nil {
			log.Print(err)
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Fatal(err)
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if namespace != "" {
				unstructuredObj.SetNamespace(namespace)
			} else if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = c.dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = c.dynamicClient.Resource(mapping.Resource)
		}

		err = dri.Delete(context.Background(), unstructuredObj.GetName(), metav1.DeleteOptions{})
		if err != nil {
			log.Print(err)
			return err
		}

		fmt.Printf("%s/%s deleted\n", unstructuredObj.GetKind(), unstructuredObj.GetName())

	}
	return nil
}

func (c *ClientGoUtils) Create(yamlPath string, namespace string, wait bool) error {
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
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			log.Print(err)
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(c.ClientSet.Discovery())
		if err != nil {
			log.Print(err)
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Fatal(err)
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if namespace != "" {
				unstructuredObj.SetNamespace(namespace)
			} else if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = c.dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = c.dynamicClient.Resource(mapping.Resource)
		}

		obj2, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			log.Print(err)
			return err
		}

		fmt.Printf("%s/%s created\n", obj2.GetKind(), obj2.GetName())

		if wait {
			err = c.WaitJobToBeReady(obj2.GetNamespace(), obj2.GetName())
			if err != nil {
				PrintlnErr("fail to wait", err)
				return err
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
		if pod.OwnerReferences != nil {
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "ReplicaSet" {
					rss, _ := c.GetReplicaSetsControlledByDeployment(context.TODO(), namespace, deployName)
					if rss != nil {
						for _, rs := range rss {
							if rs.Name == ref.Name {
								result = append(result, pod)
								continue OuterLoop
							}
						}
					}
				}
			}
		}
	}
	return result, nil

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
		if item.OwnerReferences != nil {
			for _, owner := range item.OwnerReferences {
				if owner.Name == deploymentName && item.Annotations["deployment.kubernetes.io/revision"] != "" {
					if revision, err := strconv.Atoi(item.Annotations["deployment.kubernetes.io/revision"]); err == nil {
						rsMap[revision] = item.DeepCopy()
					}
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
