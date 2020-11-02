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

package cmd

import (
	"fmt"
	"testing"
)

func TestGetFilesAndDirs(t *testing.T) {
	files, dirs, err := GetFilesAndDirs("/Users/xinxinhuang/WorkSpaces/helm/bookinfo-manifest/deployment")
	if err != nil {
		t.Error(err)
	}
	fmt.Println("files : ")
	for _, file := range files {
		fmt.Println(file)
	}
	fmt.Println("dirs : ")
	for _, dir := range dirs {
		fmt.Println(dir)
	}
}
