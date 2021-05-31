package pkg

var Option Options

type Options struct {
	Kubeconfig  string
	ServiceName string
	Namespace   string
	PortPairs   string
}
