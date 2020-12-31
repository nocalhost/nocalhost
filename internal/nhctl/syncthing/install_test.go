package syncthing

import (
	"io/ioutil"
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
