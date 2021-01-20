// +build !windows

/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flock

import (
	"errors"
	"os"
	"syscall"
)

type Flock struct {
	LockFile string
	lock     *os.File
}

func Create(file string) (f *Flock, e error) {
	if file == "" {
		e = errors.New("cannot create flock on empty path")
		return
	}
	lock, e := os.Create(file)
	if e != nil {
		return
	}
	return &Flock{
		LockFile: file,
		lock:     lock,
	}, nil
}

func (f *Flock) Release() {
	if f != nil && f.lock != nil {
		f.lock.Close()
		os.Remove(f.LockFile)
	}
}

func (f *Flock) Lock() (e error) {
	if f == nil {
		e = errors.New("cannot use lock on a nil flock")
		return
	}
	return syscall.Flock(int(f.lock.Fd()), syscall.LOCK_EX)
}

func (f *Flock) Unlock() {
	if f != nil {
		syscall.Flock(int(f.lock.Fd()), syscall.LOCK_UN)
	}
}
