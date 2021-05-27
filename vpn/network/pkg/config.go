package pkg

var OPTION Options

type Options struct {
	Kubeconfig       string
	ServiceName      string
	ServiceNamespace string
	PortPair         string
}
