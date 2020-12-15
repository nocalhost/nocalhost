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
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/term"

	"nocalhost/pkg/nhctl/log"
)

func (c *ClientGoUtils) ExecBash(namespace string, podName string, containerName string) error {
	return c.Exec(namespace, podName, containerName, []string{"sh", "-c", "clear; (bash || ash ||  sh)"})
}
func (c *ClientGoUtils) Exec(namespace string, podName string, containerName string, command []string) error {
	f := c.newFactory()

	pod, err := c.ClientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	}

	if len(containerName) == 0 {
		if len(pod.Spec.Containers) > 1 {
			fmt.Errorf("Defaulting container name to %s.\n", pod.Spec.Containers[0].Name)
		}
		containerName = pod.Spec.Containers[0].Name
	}

	t := term.TTY{
		Out: os.Stdout,
		In:  os.Stdin,
		Raw: true,
	}

	if !t.IsTerminalIn() {
		fmt.Errorf("unable to use a TTY - input is not a terminal or the right kind of file")
	}

	var sizeQueue remotecommand.TerminalSizeQueue
	if t.Raw {
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = t.MonitorSize(t.GetSize())
	}
	fn := func() error {
		rc, err := f.ToRESTConfig()
		if err != nil {
			return err
		}
		restClient, err := restclient.RESTClientFor(rc)
		if err != nil {
			return err
		}

		req := restClient.Post().
			Resource("pods").
			Name(pod.Name).
			Namespace(pod.Namespace).
			SubResource("exec")
		req.VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    false,
			TTY:       true,
		}, scheme.ParameterCodec)

		return Execute("POST", req.URL(), c.restConfig, t.In, t.Out, os.Stderr, t.Raw, sizeQueue)
	}

	if err := t.Safe(fn); err != nil {
		return err
	}
	return nil
}

func Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: terminalSizeQueue,
	})
}

func (c *ClientGoUtils) newFactory() cmdutil.Factory {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.KubeConfig = &c.kubeConfigFilePath
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	return f
}
func (c *ClientGoUtils) ApplyForCreate(files []string, namespace string, continueOnError bool) error {
	if len(files) == 0 {
		return errors.New("files must not be nil")
	}

	f := c.newFactory()
	builder := f.NewBuilder()
	validate, err := f.Validator(true)
	if err != nil {
		return err
	}
	filenames := resource.FilenameOptions{
		Filenames: files,
		Kustomize: "",
		Recursive: false,
	}
	if continueOnError {
		builder.ContinueOnError()
	}
	result := builder.Unstructured().
		Schema(validate).
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(true, &filenames).
		//LabelSelectorParam(o.Selector).
		Flatten().Do()

	if result == nil {
		return errors.New("result is nil")
	}
	if result.Err() != nil {
		return result.Err()
	}

	infos, err := result.Infos()
	if err != nil {
		return err
	}

	if len(infos) == 0 {
		return errors.New("no result info")
	}
	fmt.Printf("infos len %d \n", len(infos))
	for _, info := range infos {
		helper := resource.NewHelper(info.Client, info.Mapping)
		obj, err := helper.Create(info.Namespace, true, info.Object)
		if err != nil {
			if continueOnError {
				log.Warnf("apply manifest fail %s", err.Error())
				continue
			}
			return err
		}
		info.Refresh(obj, true)
		fmt.Printf("%s/%s created\n", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
	}

	return nil
}

func (c *ClientGoUtils) ApplyForDelete(files []string, namespace string, continueOnError bool) error {
	if len(files) == 0 {
		return errors.New("files must not be nil")
	}
	f := c.newFactory()
	builder := f.NewBuilder()
	validate, err := f.Validator(true)
	if err != nil {
		return err
	}
	filenames := resource.FilenameOptions{
		Filenames: files,
		Kustomize: "",
		Recursive: false,
	}
	if continueOnError {
		builder.ContinueOnError()
	}
	result := builder.Unstructured().
		Schema(validate).
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(true, &filenames).
		//LabelSelectorParam(o.Selector).
		Flatten().Do()

	if result == nil {
		return errors.New("result is nil")
	}
	if result.Err() != nil {
		return result.Err()
	}

	infos, err := result.Infos()
	if err != nil {
		return err
	}

	if len(infos) == 0 {
		return errors.New("no result info")
	}

	for _, info := range infos {
		helper := resource.NewHelper(info.Client, info.Mapping)
		propagationPolicy := metav1.DeletePropagationBackground
		obj, err := helper.DeleteWithOptions(info.Namespace, info.Name, &metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
		if err != nil {
			if continueOnError {
				continue
			}
			return err
		}
		info.Refresh(obj, true)
		fmt.Printf("%s/%s delete\n", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
	}

	return nil
}
