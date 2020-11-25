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

package cmds

import (
	"fmt"
	"io/ioutil"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/utils"
	"os"
	"strings"
)

func GetK8sRestClientConfig() (*restclient.Config, error) {
	home := utils.GetHomePath()
	kubeconfigPath := fmt.Sprintf("%s/.kube/config", home) // default kubeconfig
	if settings.KubeConfig != "" {
		kubeconfigPath = settings.KubeConfig
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

func GetRestClient() (*restclient.RESTClient, error) {
	k8sConfig, err := GetK8sRestClientConfig()
	if err != nil {
		fmt.Printf("%v", err)
		return nil, err
	}
	return restclient.RESTClientFor(k8sConfig)
}

func printlnErr(info string, err error) {
	fmt.Printf("[error] %s, err info: %v\n", info, err)
}

func GetFilesAndDirs(dirPth string) (files []string, dirs []string, err error) {
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, nil, err
	}

	PthSep := string(os.PathSeparator)

	for _, fi := range dir {
		if fi.IsDir() {
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			fs, ds, err := GetFilesAndDirs(dirPth + PthSep + fi.Name())
			if err != nil {
				return files, dirs, err
			}
			dirs = append(dirs, ds...)
			files = append(files, fs...)
		} else {
			ok := strings.HasSuffix(fi.Name(), ".yaml")
			if ok {
				files = append(files, dirPth+PthSep+fi.Name())
			} else if strings.HasSuffix(fi.Name(), ".yml") {
				files = append(files, dirPth+PthSep+fi.Name())
			}
		}
	}
	return files, dirs, nil
}
