/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package clientgoutils

import (
	"context"
	"fmt"
	dockerterm "github.com/moby/term"
	"github.com/nocalhost/remotecommand"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	k8sremotecommand "k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/util/term"
	"net/url"
	"nocalhost/pkg/nhctl/log"
	"os"
	"time"
)

func (c *ClientGoUtils) ExecShell(podName string, containerName string, shell string) error {
	t := term.TTY{Raw: true}
	t.MonitorSize(t.GetSize())
	// remove log eof
	k8sruntime.ErrorHandlers = k8sruntime.ErrorHandlers[1:]
	first := true
	go func() {
		for {
			cmd := shell
			if first {
				cmd = fmt.Sprintf("clear; %s", shell)
				first = false
			}
			err := c.Exec(podName, containerName, []string{"sh", "-c", cmd})
			if err == nil {
				os.Exit(0)
			}
			if e, ok := err.(exec.CodeExitError); ok && e.Code == 0 {
				os.Exit(0)
			}
			time.Sleep(time.Second * 1)
			_, err = c.ClientSet.CoreV1().Pods(c.namespace).Get(context.Background(), podName, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				log.Fatal(err)
			}
		}
	}()

	return t.Safe(func() error { select {} })
}

func (c *ClientGoUtils) Exec(podName string, containerName string, command []string) error {
	f := c.NewFactory()

	pod, err := c.ClientSet.CoreV1().Pods(c.namespace).Get(c.ctx, podName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
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

	in, out, _ := dockerterm.StdStreams()
	t := term.TTY{
		Out: out,
		In:  in,
		Raw: true,
	}

	if !t.IsTerminalIn() {
		fmt.Errorf("unable to use a TTY - input is not a terminal or the right kind of file")
	}

	var sizeQueue k8sremotecommand.TerminalSizeQueue
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
		req.VersionedParams(
			&corev1.PodExecOptions{
				Container: containerName,
				Command:   command,
				Stdin:     true,
				Stdout:    true,
				Stderr:    false,
				TTY:       true,
			}, scheme.ParameterCodec,
		)

		return Execute("POST", req.URL(), c.restConfig, t.In, t.Out, os.Stderr, t.Raw, sizeQueue)
	}

	if err := t.Safe(fn); err != nil {
		return err
	}
	return nil
}

func Execute(
	method string, url *url.URL, config *restclient.Config,
	stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue k8sremotecommand.TerminalSizeQueue,
) error {
	spdyExecutor, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return spdyExecutor.Stream(
		remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            stderr,
			Tty:               tty,
			TerminalSizeQueue: terminalSizeQueue,
		},
	)
}
