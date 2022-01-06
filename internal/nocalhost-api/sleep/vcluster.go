package sleep

import (
	"context"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"strings"

	"nocalhost/pkg/nocalhost-api/pkg/manager/vcluster"
)

func sleepVCluster(namespace string, config []byte, c kubernetes.Interface, force bool) error {
	stopChan := make(chan struct{}, 1)
	defer close(stopChan)

	vcClient, err := getVClusterClient(namespace, config, c, stopChan)
	if err != nil {
		return errors.WithStack(err)
	}
	nsList, err := vcClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, n := range nsList.Items {
		if strings.HasPrefix(n.Name, "kube-") {
			continue
		}
		if err := _asleep(config, vcClient, n.Name, force); err != nil {
			return err
		}
	}
	return nil
}

func wakeupVCluster(namespace string, config []byte, c kubernetes.Interface, force bool) error {
	stopChan := make(chan struct{}, 1)
	defer close(stopChan)

	vcClient, err := getVClusterClient(namespace, config, c, stopChan)
	if err != nil {
		return errors.WithStack(err)
	}
	nsList, err := vcClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, n := range nsList.Items {
		if strings.HasPrefix(n.Name, "kube-") {
			continue
		}
		if err := _wakeup(config, vcClient, n.Name, force); err != nil {
			return err
		}
	}
	return nil
}

func getVClusterClient(namespace string, config []byte, c kubernetes.Interface, stopChan chan struct{}) (
	kubernetes.Interface, error) {

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	vcConfig, err := vcluster.PortForwardAndGetKubeConfig(namespace, restConfig, c, stopChan)
	if err != nil {
		return nil, err
	}
	vcRestConfig, err := clientcmd.RESTConfigFromKubeConfig(vcConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	vcClient, err := kubernetes.NewForConfig(vcRestConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return vcClient, nil
}
