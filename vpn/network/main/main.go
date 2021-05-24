package main

import (
	"nocalhost/vpn/network"
)

func main() {
	//if uid := os.Geteuid(); uid != 0 {
	//	log.Fatalln("needs sudo privilege, exiting...")
	//}
	network.Start()
}
