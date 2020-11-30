package clientgoutils

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
)

type ForwardPort struct {
	LocalPort  int
	RemotePort int
}

func (c *ClientGoUtils) CreatePortForwarder(namespace string, pod string, fps []*ForwardPort) (*portforward.PortForwarder, error) {
	if fps == nil || len(fps) < 1 {
		return nil, errors.New("forward ports can not be nil")
	}

	url := c.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
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
