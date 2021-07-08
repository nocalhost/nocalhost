package app

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

func TestApplication_StopAllPortForward(t *testing.T) {
	//application, err := BuildApplication("eeee")
	//if err != nil {
	//	//cmds.printlnErr("fail to create application", err)
	//	return
	//}
	//
	//err = application.StopAllPortForward()
	//if err != nil {
	//	//cmds.printlnErr("fail to stop port-forward", err)
	//}
}

func TestForProfile2(t *testing.T) {

	go func() {
		index := 0
		for {
			profile := &AppProfileV2{}
			bytes, err := ioutil.ReadFile("profile.yaml")
			if err != nil {
				panic(err)
			}
			err = yaml.Unmarshal(bytes, profile)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%d - name is %s in profile2\n", index, profile.Name)
			if profile.Name == "" {
				for {
					fmt.Println("Retry .... in profile2")
					bytes, err = ioutil.ReadFile("profile.yaml")
					if err != nil {
						panic(err)
					}
					err = yaml.Unmarshal(bytes, profile)
					if err != nil {
						panic(err)
					}
					fmt.Printf("%d - name is %s in profile2\n", index, profile.Name)
					if profile.Name == "" {
						fmt.Println("Failed retry ... in profile 2")
					} else {
						fmt.Println("Success retry ...in profile 2")
						time.Sleep(5 * time.Second)
						os.Exit(0)
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
			err = ioutil.WriteFile("profile.yaml", bytes, 0644)
			if err != nil {
				fmt.Println(err.Error())
				panic(err)
			}
			time.Sleep(100 * time.Millisecond)
			index++
		}
	}()

	time.Sleep(1 * time.Hour)
}

func TestForProfile(t *testing.T) {

	fmt.Println("Reading")
	go func() {
		index := 0
		for {
			profile := &AppProfileV2{}
			bytes, err := ioutil.ReadFile("profile.yaml")
			if err != nil {
				panic(err)
			}
			err = yaml.Unmarshal(bytes, profile)
			if err != nil {
				panic(err)
			}
			//fmt.Printf("%d - name is %s in profile\n", index, profile.Name)
			if profile.Name == "" {
				for {
					fmt.Println("Retry .... ")
					bytes, err = ioutil.ReadFile("profile.yaml")
					if err != nil {
						panic(err)
					}
					err = yaml.Unmarshal(bytes, profile)
					if err != nil {
						panic(err)
					}
					fmt.Printf("%d - name is %s in profile\n", index, profile.Name)
					if profile.Name == "" {
						fmt.Println("Failed retry ...")
					} else {
						fmt.Println("Success retry ...")
						time.Sleep(5 * time.Second)
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
			}

			err = ioutil.WriteFile("profile.yaml", bytes, 0644)
			if err != nil {
				fmt.Println(err.Error())
				panic(err)
			}
			time.Sleep(10 * time.Millisecond)
			index++
		}
	}()

	time.Sleep(1 * time.Hour)
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
