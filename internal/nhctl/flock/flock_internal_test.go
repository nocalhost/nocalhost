package flock

import (
	"io/ioutil"
	"os"
	"testing"
)

func Test(t *testing.T) {
	tmpFileFh, err := ioutil.TempFile(os.TempDir(), "go-flock-")
	tmpFileFh.Close()
	tmpFile := tmpFileFh.Name()
	os.Remove(tmpFile)

	lock := New(tmpFile)
	locked, err := lock.TryLock()
	if locked == false || err != nil {
		t.Fatalf("failed to lock: locked: %t, err: %v", locked, err)
	}

	newLock := New(tmpFile)
	locked, err = newLock.TryLock()
	if locked != false || err != nil {
		t.Fatalf("should have failed locking: locked: %t, err: %v", locked, err)
	}

	if newLock.fh != nil {
		t.Fatal("file handle should have been released and be nil")
	}
}
