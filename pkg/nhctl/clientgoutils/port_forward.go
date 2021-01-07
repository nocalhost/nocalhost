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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type ForwardPort struct {
	LocalPort  int
	RemotePort int
}

func (c *ClientGoUtils) CreatePortForwarder(pod string, fps []*ForwardPort) (*portforward.PortForwarder, error) {
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

	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	out := new(bytes.Buffer)

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
		out)
	if err != nil {
		return nil, err
	}
	return pf, nil
}
