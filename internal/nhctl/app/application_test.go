package app

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"sync"
	"testing"
	"time"
)

func TestApplication_ReplaceString(t *testing.T) {
	oldStr := "codingcorp-docker.pkg.coding.net/dev-images/golang"
	re3, _ := regexp.Compile("codingcorp-docker.pkg.coding.net")
	fmt.Println(re3.ReplaceAllString(oldStr, "nocalhost-docker.pkg.coding.net"))
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
