package main

import (
	"flag"
	"nocalhost/vpn/network/pkg"
)

func parseParam() {
	flag.StringVar(&pkg.OPTION.Kubeconfig, "kubeconfig", "", "your k8s cluster kubeconfig path")
	flag.StringVar(&pkg.OPTION.ServiceName, "name", "", "service name and deployment name, should be same")
	flag.StringVar(&pkg.OPTION.ServiceNamespace, "namespace", "", "service namespace")
	flag.StringVar(&pkg.OPTION.PortPair, "expose", "", "port pair, remote-port:local-port, such as: service-port1:local-port1,service-port2:local-port2...")
	flag.Parse()
}

func main() {
	//if uid := os.Geteuid(); uid != 0 {
	//	log.Fatalln("needs sudo privilege, exiting...")
	//}
	if err := pkg.PreCheck(); err != nil {
		panic(err)
	}
	parseParam()
	pkg.Start(pkg.OPTION)
}
