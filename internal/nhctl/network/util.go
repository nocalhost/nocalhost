package network

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/portforward"
	clientgowatch "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/transport/spdy"
	"log"
	"net/http"
	"nocalhost/internal/nhctl/model"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

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

func portForwardPod(podName string, namespace string, port int, readyChan, stopChan chan struct{}) error {
	url := clientset.CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward").
		URL()
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		log.Println(err)
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
	out := new(bytes.Buffer)
	p := []string{fmt.Sprintf("%d:%d", port, 22)}
	forwarder, err := portforward.New(dialer, p, stopChan, readyChan, ioutil.Discard, out)
	if err != nil {
		log.Println(err)
		return err
	}

	if err = forwarder.ForwardPorts(); err != nil {
		panic(err)
	}
	return nil
}

func portForwardService(options model.Options, localSshPort int, okChan chan struct{}) {
	cmd := exec.
		CommandContext(
			context.TODO(),
			"kubectl",
			"port-forward",
			"service/"+DNSPOD,
			strconv.Itoa(localSshPort)+":22",
			"--namespace",
			options.Namespace,
			"--kubeconfig",
			options.Kubeconfig)
	_, _, err := RunWithRollingOut(cmd, func(s string) bool {
		if strings.Contains(s, "Forwarding from") {
			okChan <- struct{}{}
			return true
		}
		return false
	})
	if err != nil {
		log.Println(err)
	}
}

func scaleDeploymentReplicasTo(options model.Options, replicas int32) {
	_, err := clientset.AppsV1().Deployments(options.Namespace).
		UpdateScale(context.TODO(), options.ServiceName, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      options.ServiceName,
				Namespace: options.Namespace,
			},
			Spec: autoscalingv1.ScaleSpec{Replicas: replicas},
		}, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("update deployment: %s's replicas to %d failed, error: %v\n", options.ServiceName, replicas, err)
	}
}
