package clientgoutils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	batchv1 "k8s.io/api/batch/v1"
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
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type ClientGoUtils struct {
	kubeConfigFilePath string
	//restConfig *restclient.Config
	ClientSet          *kubernetes.Clientset
	dynamicClient      dynamic.Interface//
	//RestClient         *restclient.RESTClient
}


func NewClientGoUtils(kubeConfigPath string) (*ClientGoUtils, error) {
	var (
		err error
		restConfig         *restclient.Config
	)


	if kubeConfigPath == "" {  // use kubectl default config
		kubeConfigPath = fmt.Sprintf("%s/.kube/config", getHomePath())
	}
	client := &ClientGoUtils{
		kubeConfigFilePath: kubeConfigPath,
	}

	if restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil{
		printlnErr("fail to build rest config", err)
		return nil, err
	}

	if client.ClientSet, err = kubernetes.NewForConfig(restConfig); err != nil {
		printlnErr("fail to get clientset", err)
		return nil, err
	}

	if client.dynamicClient, err = dynamic.NewForConfig(restConfig); err != nil {
		printlnErr("fail to get dynamicClient", err)
		return nil, err
	}

	return client, nil
}

func (c *ClientGoUtils) getRestConfig() (*restclient.Config, error){
	return clientcmd.BuildConfigFromFlags("", c.kubeConfigFilePath)
}

func (c *ClientGoUtils) Create(yamlPath string, namespace string, wait bool) error{
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

		obj2, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
		if  err != nil {
			log.Print(err)
			return err
		}

		fmt.Printf("%s/%s created\n",obj2.GetKind(), obj2.GetName())

		if wait {
			err = c.WaitJobToBeReady(obj2.GetNamespace(), obj2.GetName())
			if err != nil {
				printlnErr("fail to wait", err)
				return err
			}
		}
	}

	return nil
}

func (c *ClientGoUtils)GetDiscoveryClient() (*discovery.DiscoveryClient, error){
	config, err := c.getRestConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(config)
}

func (c *ClientGoUtils)Discovery(){
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

func (c *ClientGoUtils)waitUtilReady(kind string, namespace string, name string,uns *unstructured.Unstructured) error {
	//
	//var err error
	//restClient := c.RestClient
	//switch kind {
	//case "Job":
	//	fmt.Println("waiting job")
	//	restClient, err = c.GetRestClient(BatchV1)
	//case "Pod": // only wait for job and pod
	//	fmt.Println("waiting for " + kind)
	//default:
	//	fmt.Println("no waiting for " + kind)
	//	return nil
	//}


	//selector, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", name))
	//if err != nil {
	//	return err
	//}

	//lw := cachetools.NewListWatchFromClient(restClient, "jobs", namespace, selector)
	//ctx, cancel := watchtools.ContextWithOptionalTimeout(context.Background(), time.Minute*5)
	//defer cancel()
	//_, err = watchtools.UntilWithSync(ctx, lw, uns, nil, func(e watch.Event) (bool, error) {
	//	switch e.Type {
	//	case watch.Added, watch.Modified:
	//		// For things like a secret or a config map, this is the best indicator
	//		// we get. We care mostly about jobs, where what we want to see is
	//		// the status go into a good state. For other types, like ReplicaSet
	//		// we don't really do anything to support these as hooks.
	//		switch kind {
	//		case "Job":
	//			return waitForJob(e.Object, name)
	//		//case "Pod":
	//		//	return waitForPodSuccess(e.Object, resourceName)
	//		}
	//		return true, nil
	//	case watch.Deleted:
	//		fmt.Printf("Deleted event for %s", name)
	//		return true, nil
	//	case watch.Error:
	//		// Handle error and return with an error.
	//		fmt.Printf("Error event for %s", name)
	//		return true, errors.New("failed to deploy " + name)
	//	default:
	//		return false, nil
	//	}
	//})

	return nil
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



