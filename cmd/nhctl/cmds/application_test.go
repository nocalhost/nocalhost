package cmds

import (
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestApplication_StopAllPortForward(t *testing.T) {
	application, err := NewApplication("eeee")
	if err != nil {
		printlnErr("fail to create application", err)
		return
	}

	err = application.StopAllPortForward()
	if err != nil {
		printlnErr("fail to stop port-forward", err)
	}
}

func TestForTest(t *testing.T) {
	map2 := v1.ConfigMap{}
	map2.Name = "nocalhost-depends-do-not-overwrite-"
	//map2.Data
}

func TestWaitGroup(t *testing.T) {

	wg := sync.WaitGroup{}
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(ii int) {
			s := rand.Intn(10) + 1
			fmt.Printf("%d sleep %d seconds\n", ii, s)
			time.Sleep(time.Second * time.Duration(s))
			fmt.Printf("%d exit\n", ii)
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestIoUtilWriteFile(t *testing.T) {
	context := `
aaaa
`
	err := ioutil.WriteFile("aa.txt", []byte(context), 0755)
	if err != nil {
		panic(err)
	}
}
