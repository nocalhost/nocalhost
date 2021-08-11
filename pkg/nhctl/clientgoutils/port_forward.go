/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"net/http"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type ForwardPort struct {
	LocalPort  int
	RemotePort int
}

func (c *ClientGoUtils) Forward(pod string, localPort, remotePort int, readyChan, stopChan chan struct{}, g genericclioptions.IOStreams) error {
	pf := ForwardPort{
		LocalPort:  localPort,
		RemotePort: remotePort,
	}
	fd, err := c.CreatePortForwarder(pod, []*ForwardPort{&pf}, readyChan, stopChan, g)
	if err != nil {
		return err
	}
	return errors.Wrap(fd.ForwardPorts(), "")
}

func (c *ClientGoUtils) CreatePortForwarder(pod string, fps []*ForwardPort, readyChan, stopChan chan struct{}, g genericclioptions.IOStreams) (*portforward.PortForwarder, error) {
	if fps == nil || len(fps) < 1 {
		return nil, errors.New("forward ports can not be nil")
	}

	url := c.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(c.namespace).
		Name(pod).
		SubResource("portforward").URL()

	transport, upgrader, err := spdy.RoundTripperFor(c.restConfig)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

	if readyChan == nil {
		readyChan = make(chan struct{})
	}
	if stopChan == nil {
		stopChan = make(chan struct{})
	}
	//out := new(bytes.Buffer)

	ports := make([]string, 0)
	for _, fp := range fps {
		ports = append(ports, fmt.Sprintf("%d:%d", fp.LocalPort, fp.RemotePort))
	}

	pf, err := portforward.NewOnAddresses(
		dialer,
		[]string{"localhost"},
		ports,
		stopChan,
		readyChan,
		ioutil.Discard,
		g.Out,
	)
	if err != nil {
		return nil, err
	}
	return pf, nil
}
