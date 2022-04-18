/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/cmd/portforward"
	"net/http"
	"net/url"
)

type ForwardPort struct {
	LocalPort  int
	RemotePort int
}

type PortForwardFlags struct {
	ResourcesName string // eg: "pod/pod-01", "deployment/d1"
	ForwardPort   string // eg: 1110:1110, :1234
	Streams       genericclioptions.IOStreams
	StopChannel   chan struct{}
	ReadyChannel  chan struct{}
}

type ClientgoPortForwarder struct {
	genericclioptions.IOStreams
	pw *PortForwarder
}

func (f ClientgoPortForwarder) GetPorts() ([]ForwardedPort, error) {
	if f.pw != nil {
		return f.pw.GetPorts()
	}
	return nil, errors.New("port is nil")
}

func (f *ClientgoPortForwarder) ForwardPorts(method string, url *url.URL, opts portforward.PortForwardOptions) error {
	transport, upgrader, err := spdy.RoundTripperFor(opts.Config)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, method, url)
	fw, err := NewOnAddresses(dialer, opts.Address, opts.Ports, opts.StopChannel, opts.ReadyChannel, f.Out, f.ErrOut)
	if err != nil {
		return err
	}
	f.pw = fw
	return fw.ForwardPorts()
}

func (c *ClientGoUtils) ForwardPortForwardByPod(pod string, localPort, remotePort int, readyChan, stopChan chan struct{}, g genericclioptions.IOStreams) error {
	client, err := c.NewFactory().RESTClient()
	if err != nil {
		return err
	}
	req := client.Post().
		Resource("pods").
		Namespace(c.namespace).
		Name(pod).
		SubResource("portforward")
	forwarder := &ClientgoPortForwarder{IOStreams: g}
	return forwarder.ForwardPorts("POST", req.URL(), portforward.PortForwardOptions{
		Config:       c.restConfig,
		Address:      []string{"0.0.0.0"},
		Ports:        []string{fmt.Sprintf("%d:%d", localPort, remotePort)},
		StopChannel:  stopChan,
		ReadyChannel: readyChan,
	})
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

func (c *ClientGoUtils) CreatePortForwarder(pod string, fps []*ForwardPort, readyChan, stopChan chan struct{}, g genericclioptions.IOStreams) (*PortForwarder, error) {
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

	pf, err := NewOnAddresses(
		dialer,
		[]string{"0.0.0.0"},
		ports,
		stopChan,
		readyChan,
		g.Out,
		g.Out,
	)
	if err != nil {
		return nil, err
	}
	return pf, nil
}
