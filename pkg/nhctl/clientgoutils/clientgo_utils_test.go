package clientgoutils

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewClientGoUtils(t *testing.T) {
	client,_ := NewClientGoUtils("")  // /Users/xinxinhuang/.kube

	depList, err := client.ClientSet.AppsV1().Deployments("xinxinhuang").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		printlnErr("err", err)
		return
	}

	for _,v := range depList.Items {
		fmt.Println(v.Name)
	}
}

func TestClientGoUtils_Create(t *testing.T) {
	client,_ := NewClientGoUtils("/Users/xinxinhuang/.kube/admin-config")
	err := client.Create("/Users/xinxinhuang/WorkSpaces/helm/bookinfo-manifest/deployment/templates/pre-install/print-num-job-01.yaml","demo3", true)
	if err != nil {
		printlnErr("failed : ", err)
	}
}

func TestClientGoUtils_Discovery(t *testing.T) {
	client, _:= NewClientGoUtils("")
	client.Discovery()
}

//func TestClientGoUtils_GetRestClient(t *testing.T) {
//	client := NewClientGoUtils("/Users/xinxinhuang/.kube/admin-config")
//	restClient , err := client.GetRestClient(BatchV1)
//	if err != nil {
//		printlnErr("failed !!!" , err)
//		return
//	}
//	fmt.Println(restClient)
//}