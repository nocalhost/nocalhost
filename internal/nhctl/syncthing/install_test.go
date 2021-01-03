package syncthing

import (
	"fmt"
	"io/ioutil"
	"nocalhost/internal/nhctl/nocalhost"
	"path/filepath"
	"testing"
)

func TestDownLoad(t *testing.T) {
	tmpDir, _ := ioutil.TempDir("", "")

	mockedSyncthing := &Syncthing{
		BinPath: tmpDir,
	}

	err := mockedSyncthing.DownloadSyncthing(SyncthingVersion)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNeedToDownload(t *testing.T) {
	mockedSyncthing := &Syncthing{
		BinPath: filepath.Join(nocalhost.GetSyncThingBinDir(), "syncthing"),
	}

	needToDownload := mockedSyncthing.NeedToDownloadSpecifyVersion("")
	fmt.Printf("need to download: %t  ", needToDownload)
}
