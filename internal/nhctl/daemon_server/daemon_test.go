package daemon_server

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"net"
	"nocalhost/internal/nhctl/appmeta"
	profile2 "nocalhost/internal/nhctl/profile"
	"testing"
)

func TestUnMar(t *testing.T) {
	meta := &appmeta.ApplicationMeta{}

	meta.Config = &profile2.NocalHostAppConfigV2{}
	meta.Config.ApplicationConfig = profile2.ApplicationConfig{}

	meta.Config.ApplicationConfig.HelmVals = map[string]interface{}{
		"service": map[interface{}]interface{}{"port": "9082"},
		"bookinfo": map[string]interface{}{
			"deploy": map[string]interface{}{
				"resources": map[interface{}]interface{}{"limit": map[interface{}]interface{}{"cpu": "700m"}},
			},
		},
	}

	if m, err := yaml.Marshal(meta); err != nil {
		t.Error(err)
	} else {
		println(string(m))

		metas := &appmeta.ApplicationMeta{}
		_ = yaml.Unmarshal(m, metas)

		if massss, err := json.Marshal(metas); err != nil {
			t.Error(err)
		} else {
			println(string(massss))
		}
	}
}

func TestErr(t *testing.T) {
	fmt.Println("before")
	err := errTest()
	fmt.Printf("after: %v\n", err)
}

func errTest() error {
	var err error
	num, err := errTest2("err01")
	fmt.Printf("%v\n", &num)
	if num != 0 {
		fmt.Println("a")
	}
	defer func() {
		err = errors.Wrap(err, "")
		fmt.Printf("err handling: %v\n", err)
	}()

	_, err = errTest2("err02")
	err = errors.Wrap(err, "")
	fmt.Printf("%v\n", err)
	//fmt.Printf("%v\n", &num)

	return err
}

func errTest2(info string) (int, error) {
	//_, err := strconv.ParseBool("aaa")
	return 0, errors.New(info)
}

func TestPortForwardManager_StartPortForwardGoRoutine(t *testing.T) {
	address := fmt.Sprintf("0.0.0.0:%d", 89)
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		panic(err)
	}
	_, err = listener.Accept()
	if err != nil {
		panic(err)
	}
	fmt.Println("Listen")
	_ = listener.Close()
}

func TestParseErrToForwardPort(t *testing.T) {
	p, err := ParseErrToForwardPort("error creating error stream for port 20153 -> 20153: Timeout occured")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", p)
}
