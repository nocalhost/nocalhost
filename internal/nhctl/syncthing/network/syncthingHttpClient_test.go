package syncthing

import (
	"encoding/json"
	"fmt"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"testing"
	"time"
)

// those test case only valid while dev mode enabled.
var client = req.NewSyncthingHttpClient(
	"127.0.0.1:64500",
	"nocalhost",
	"MDPJNTF-OSPJC65-LZNCQGD-3AWRUW6-BYJULSS-GOCA2TU-5DWWBNC-TKM4VQ5",
	"nh-1",
)

func TestSystemConnections(t *testing.T) {
	connections, err := client.SystemConnections()
	if err != nil {
		t.Fatal(err)
	}

	if connections {
		fmt.Print("connect ~")
	} else {
		fmt.Print("no connect ~")
	}
}

func TestGetFolderStatus(t *testing.T) {
	status, err := client.FolderStatus()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("State changed in %v", status.StateChanged)
}

// {GlobalBytes:46490586 GlobalDeleted:16 GlobalFiles:37 InSyncBytes:46490586 InSyncFiles:37 Invalid: LocalBytes:46490586
//  LocalDeleted:16 LocalFiles:37 NeedBytes:0 NeedFiles:0 State:scanning StateChanged:2021-01-18 19:39:03.684895 +0800 CST Version:205}

func TestCompletion(t *testing.T) {
	comp, err := client.Completion()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v", comp)
}

func TestOverrideFolder(t *testing.T) {
	err := client.FolderOverride()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSyncthingStatus(t *testing.T) {
	for true {
		time.Sleep(time.Duration(500) * time.Millisecond)
		status := client.GetSyncthingStatus()
		marshal, _ := json.Marshal(status)
		fmt.Printf("%+v\n", string(marshal))
	}
}
