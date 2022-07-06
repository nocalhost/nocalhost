package watcher

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	_const "nocalhost/internal/nhctl/const"
	utils2 "nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"testing"
)

func TestSimpleWatcher(t *testing.T) {
	utils, _ := clientgoutils.NewClientGoUtils(os.Getenv("KUBECONFIG"), "f")

	printer := utils2.NewPrinter(
		func(s string) {
			log.Infof(s)
		},
	)

	c := NewSimpleWatcher(
		utils,
		"pod",
		metav1.ListOptions{
			LabelSelector: kblabels.Set{"app": "authors"}.AsSelector().String(),
		},
		func(key string, object interface{}, quitChan chan struct{}) {
			if us, ok := object.(*unstructured.Unstructured); ok {
				containerStatusForDevPod(
					us, func(status string, err error) {
						printer.ChangeContent(status)
					},
				)
			}
		},
		nil,
	)

	log.Info("Waiting")
	<-c
}

func TestEventWatcher(t *testing.T) {
	os.Setenv("KUBECONFIG", "/Users/anur/.kube/config")

	utils, _ := clientgoutils.NewClientGoUtils(os.Getenv("KUBECONFIG"), "f")

	c := NewSimpleWatcher(
		utils,
		"event",
		metav1.ListOptions{},
		func(key string, object interface{}, quitChan chan struct{}) {
			if us, ok := object.(*unstructured.Unstructured); ok {
				var event corev1.Event
				if err := runtime.DefaultUnstructuredConverter.
					FromUnstructured(us.UnstructuredContent(), &event); err != nil {
					return
				}

				if event.Type == "Normal" {
					return
				}

				println()
			}
		},
		nil,
	)

	log.Info("Waiting")
	<-c
}

func containerStatusForDevPod(maybePod *unstructured.Unstructured, consumeFun func(status string, err error)) {
	var pod corev1.Pod
	if err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(maybePod.UnstructuredContent(), &pod); err != nil {
		consumeFun("", err)
		return
	}

	if maybePod.GetDeletionTimestamp() != nil {
		return
	}

	head := fmt.Sprintf("Pod %s now %s", pod.Name, pod.Status.Phase)
	msg := ""
	if len(pod.Status.Conditions) > 0 {
		for _, condition := range pod.Status.Conditions {
			if condition.Status != "True" {
				head += "\n - Condition: " + condition.Reason + ", " + condition.Message
			}
		}
	}

	annotations := maybePod.GetAnnotations()
	devContainerName, ok := annotations[_const.NocalhostDevContainerAnnotations]
	if !ok {
		devContainerName = _const.NocalhostDefaultDevContainerName
	}

	for _, status := range pod.Status.ContainerStatuses {

		if status.Name != devContainerName && status.Name != _const.NocalhostDefaultDevSidecarName {
			continue
		}

		msg += "\n > Container: " + status.Name

		if status.State.Running != nil {
			msg += " is Running"
		} else if status.State.Terminated != nil {
			msg += " is Terminated"
		} else if status.State.Waiting != nil {
			msg += " is Waiting, Reason: " + status.State.Waiting.Reason
			if status.State.Waiting.Message != "" {
				msg += " Msg: " + status.State.Waiting.Message
			}
		}
	}

	if msg != "" {
		consumeFun(head+msg+"\n", nil)
	}
}
