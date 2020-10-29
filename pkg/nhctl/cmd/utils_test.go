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
	for _, dir := range dirs{
		fmt.Println(dir)
	}
}
