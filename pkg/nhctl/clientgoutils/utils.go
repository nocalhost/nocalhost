/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clientgoutils

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientGoUtils struct {
	kubeConfigFilePath string
	ClientSet          *kubernetes.Clientset
	dynamicClient      dynamic.Interface //
	RestClient         *restclient.RESTClient
}

func NewClientGoUtils(kubeConfigPath string) (*ClientGoUtils, error) {
	var (
		err        error
		restConfig *restclient.Config
	)

	if kubeConfigPath == "" { // use kubectl default config
		kubeConfigPath = fmt.Sprintf("%s/.kube/config", getHomePath())
	}
	client := &ClientGoUtils{
		kubeConfigFilePath: kubeConfigPath,
	}

	if restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil {
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

func (c *ClientGoUtils) getRestConfigFromKubeConfig() (*restclient.Config, error) {
	return clientcmd.BuildConfigFromFlags("", c.kubeConfigFilePath)
}

func (c *ClientGoUtils) Create(yamlPath string, namespace string, wait bool) error {
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
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace(namespace)
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
				printlnErr("fail to wait", err)
				return err
			}
		}
	}

	// 通过 manifest 内容创建 []*resource.Info, 需要 factoryImpl.clientGetter
	// Builder 用来接收命令行传过来的参数，并把它们转成一系列资源，然后通过Vistor接口来遍历
	// resource.NewBuilder(f.clientGetter).  // clientGetter 就是 genericclioptions.ConfigFlags， 实际上是作为 resource.RESTClientGetter 使用的
	// ContinueOnError().
	// NamespaceParam(namespace string).  // namespace 就可以，设置namespace参数
	// DefaultNamespace().  // 如果没有指定namespace，则自动设置namespace为NamespaceParam中设置的参数
	// Unstructured(). // update 一下 builder，让它会请求和发送 unstructured 的对象， unstructured 对象保存服务端发送过来的所有字段在一个map中，
	//基于对象的JSON结构，这意味着当 client 去读的时候，没有数据会丢失。一般只在内部使用这种模式
	// Schema().
	// Stream(reader, "").  //  解析 reader 成一个对象吧？？？
	// reader 是一个 manifest 的 buffer！
	// Do().      // 		返回一个 Result
	// Infos() 返回所有资源对象的info
	return nil
}

func (c *ClientGoUtils) waitUtilReady(kind string, namespace string, name string, uns *unstructured.Unstructured) error {
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
