package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"k8s.io/client-go/util/homedir"
	"nocalhost/test/util"
	"os"
	"path/filepath"
)

type TkeOption struct {
	Filename string
}

var tkeOption TkeOption

func init() {
	flag.StringVar(&tkeOption.Filename, "filename", filepath.Join(homedir.HomeDir(), "CLUSTER_ID"), "")
	flag.Parse()
}

func main() {
	if len(os.Args) <= 1 {
		logrus.Infof("please provide a command, supported value are create, destroy")
	}
	if os.Args[1] == "create" {
		t, err := CreateK8s()
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("create cluster: %s secuessfully", t.clusterId)
		if err = writeClusterIdToFile(tkeOption.Filename, t.clusterId); err != nil {
			logrus.Fatalf("write cluster id failed, cluster id is %s", t.clusterId)
		}
		logrus.Infof("write cluster to file: %s sucessfully", tkeOption.Filename)
	} else if os.Args[1] == "destroy" {
		file, err := readClusterIdFromFile(tkeOption.Filename)
		if err != nil {
			panic(err)
		}
		t := NewTask(os.Getenv(util.SecretId), os.Getenv(util.SecretKey))
		t.clusterId = file
		DeleteTke(t)
		logrus.Infof("delete cluster: %s secuessfully", file)
	} else {
		logrus.Infof("provide command: %s is not support, supported value are create, destroy", os.Args[1])
	}
}

func writeClusterIdToFile(filename, clusterId string) error {
	return ioutil.WriteFile(filename, []byte(clusterId), fs.ModePerm)
}
func readClusterIdFromFile(filename string) (string, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(file), err
}
