package nhctl

import (
	"fmt"
	"nocalhost/test/util"
	"testing"
	"time"
)

func TestInstallNhctl(t *testing.T) {
	InstallNhctl("1f820388196a2bc57a7d118d46c40e9f99c8c119")
}

func TestInit(t *testing.T) {
	go Init()
	for {
		select {
		case status := <-StatusChan:
			if status == 0 {
				fmt.Println("ok")
				//StopChan <- 0
			} else {
				fmt.Printf("not ok")
			}
		}
	}
}

func TestCommandWait(t *testing.T) {
	go util.WaitForCommandDone("kubectl wait --for=condition=Delete pods/test-74cb446f4b-hmp8g -n test")
	time.Sleep(7 * time.Second)
}

func TestDev(t *testing.T) {
	moduleName := "details"
	Dev(moduleName)
	Sync(moduleName)
	End(moduleName)
}
