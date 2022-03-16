/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"fmt"
	"github.com/fatih/color"
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
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

func (c *ClientGoUtils) ExecShell(podName string, containerName string, shell, banner string) error {
	t := term.TTY{Raw: true, Out: os.Stdout}
	t.MonitorSize(t.GetSize())

	if banner != "" {
		cc := color.New(color.BgGreen).Add(color.FgBlack).Add(color.Bold)
		t.Out.Write([]byte(cc.Sprintln(banner)))
	}

	for i := 0; i < len(k8sruntime.ErrorHandlers); i++ {
		fn := runtime.FuncForPC(reflect.ValueOf(k8sruntime.ErrorHandlers[i]).Pointer()).Name()
		if strings.Contains(fn, "logError") {
			k8sruntime.ErrorHandlers = append(k8sruntime.ErrorHandlers[:i], k8sruntime.ErrorHandlers[i+1:]...)
		}
	}

	first := true
	errChan := make(chan error, 0)
	go func() {
		var err error
		defer func() {
			errChan <- err
		}()
		for {
			cmd := shell
			if first {
				cmd = fmt.Sprintf(`%s`, shell)
				first = false
			}
			err = c.Exec(podName, containerName, []string{"sh", "-c", cmd})
			if err == nil {
				return
			}
			if e, ok := err.(exec.CodeExitError); ok && e.Code == 0 {
				//os.Exit(0)
				return
			}
			time.Sleep(time.Second * 1)
			_, err = c.ClientSet.CoreV1().Pods(c.namespace).Get(context.Background(), podName, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				return
			}
		}
	}()

	return t.Safe(func() error {
		return <-errChan
	})
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
	ops := remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: terminalSizeQueue,
	}
	return spdyExecutor.Stream(ops)
}
