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
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	cachetools "k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/tools"
	"nocalhost/pkg/nhctl/utils"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type InstallFlags struct {
	*EnvSettings
	//Name                 string
	Url                  string // resource url
	AppType              string
	ResourcesDir         string
	HelmValueFile        string
	PreInstallConfigPath string
	ForceInstall         bool
}

var installFlags = InstallFlags{
	EnvSettings: settings,
}

func init() {
	installCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	installCmd.Flags().StringVarP(&installFlags.Url, "url", "u", "", "resource url")
	installCmd.Flags().StringVarP(&installFlags.ResourcesDir, "dir", "d", "", "the dir of helm package or manifest")
	installCmd.Flags().StringVarP(&installFlags.HelmValueFile, "", "f", "", "helm's Value.yaml")
	installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "helm", "app type: helm or manifest")
	installCmd.Flags().BoolVar(&installFlags.ForceInstall, "force", installFlags.ForceInstall, "force install")
	installCmd.Flags().StringVarP(&installFlags.PreInstallConfigPath, "pre-install", "p", "", "resources to be installed before application install, should be a yaml file path")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install [NAME]",
	Short: "install k8s application",
	Long:  `install k8s application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		//if installFlags.Name == "" {
		//	fmt.Println("error: please use --name to specify the name of nocalhost application")
		//	return
		//}
		if installFlags.Url == "" {
			fmt.Println("error: please use -u to specify url of git")
			return
		}
		fmt.Println("install application...")
		InstallApplication(args[0])
	},
}

func InstallApplication(applicationName string) {

	var (
		resourcesPath string
		err           error
	)

	// create application dir
	applicationDir := GetHomePath() + "/.nhctl/" + "application/" + applicationName
	if _, err = os.Stat(applicationDir); err != nil {
		if os.IsNotExist(err) {
			debug("%s not exists, create application dir", applicationDir)
			utils.Mush(os.Mkdir(applicationDir, 0755))
			utils.Mush(DownloadApplicationToNhctlHome(applicationDir))
		} else {
			panic(err)
		}
	} else if !installFlags.ForceInstall {
		fmt.Printf("%s already exists, please use --force to force it to be reinstalled\n", applicationName)
		return
	} else if installFlags.ForceInstall {
		fmt.Printf("force to reinstall %s\n", applicationName)
		utils.Mush(os.RemoveAll(applicationDir))
		utils.Mush(os.Mkdir(applicationDir, 0755))
		utils.Mush(DownloadApplicationToNhctlHome(applicationDir))
	}

	if installFlags.ResourcesDir != "" {
		resourcesPath = fmt.Sprintf("%s%c%s", applicationDir, os.PathSeparator, installFlags.ResourcesDir)
	} else {
		resourcesPath = applicationDir
	}
	debug("resources path is %s\n", resourcesPath)
	if installFlags.AppType == "helm" {
		params := []string{"upgrade", "--install", "--wait", applicationName, resourcesPath, "--debug"}
		if nameSpace != "" {
			params = append(params, "-n", nameSpace)
		}
		if settings.KubeConfig != "" {
			params = append(params, "--kubeconfig", settings.KubeConfig)
		}
		fmt.Println("install helm application, this may take several minutes, please waiting...")
		output, err := tools.ExecCommand(nil, false, "helm", params...)
		if err != nil {
			printlnErr("fail to install helm app", err)
			return
		}
		debug(output)
		fmt.Printf(`helm app installed, use "helm list -n %s" to get the information of the helm release`+"\n", nameSpace)
	} else if installFlags.AppType == "manifest" {
		excludeFiles := make([]string, 0)
		if installFlags.PreInstallConfigPath != "" {
			excludeFiles, err = PreInstall(resourcesPath)
			if err != nil {
				panic(err)
			}
		}
		// install manifest recursively
		InstallManifestRecursively(resourcesPath, excludeFiles)
	} else {
		fmt.Println("unsupported type")
	}
}

func DownloadApplicationToNhctlHome(homePath string) error {
	var (
		err        error
		gitDirName string
	)

	if strings.HasPrefix(installFlags.Url, "https") || strings.HasPrefix(installFlags.Url, "git") || strings.HasPrefix(installFlags.Url, "http") {
		if strings.HasSuffix(installFlags.Url, ".git") {
			gitDirName = installFlags.Url[:len(installFlags.Url)-4]
		} else {
			gitDirName = installFlags.Url
		}
		debug("git dir : " + gitDirName)
		strs := strings.Split(gitDirName, "/")
		gitDirName = strs[len(strs)-1] // todo : for default application anme
		// clone git
		//gitPath := fmt.Sprintf("%s%c%s", homePath, os.PathSeparator, gitDirName)
		_, err = tools.ExecCommand(nil, true, "git", "clone", installFlags.Url, homePath)
		if err != nil {
			printlnErr("fail to clone git", err)
			return err
		}
	} else { // todo: for no git url
		fmt.Println("installing ")
		return nil
	}
	return nil
}

type PreInstallConfig struct {
	Items []Item `yaml:"items"`
}

type Item struct {
	Path   string `yaml:"path"`
	Weight string `yaml:"weight"`
}

type ComparableItems []Item

func (a ComparableItems) Len() int      { return len(a) }
func (a ComparableItems) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ComparableItems) Less(i, j int) bool {
	iW, err := strconv.Atoi(a[i].Weight)
	if err != nil {
		iW = 0
	}

	jW, err := strconv.Atoi(a[j].Weight)
	if err != nil {
		jW = 0
	}
	return iW < jW
}

func InstallManifestRecursively(dir string, excludeFiles []string) error {

	files, _, err := GetFilesAndDirs(dir)
	if err != nil {
		return err
	}

outer:
	for _, file := range files {
		for _, ex := range excludeFiles {
			if ex == file {
				fmt.Println("ignore file : " + file)
				continue outer
			}
		}
		fmt.Println("create " + file)
		clientUtil, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, 0)
		if err != nil {
			return err
		}
		clientUtil.Create(file, nameSpace, false)
	}
	return err
}

func PreInstall(basePath string) ([]string, error) {
	var (
		configFilePath string
	)

	fmt.Println("run pre-install....")
	//  读取一个yaml文件
	pConf := &PreInstallConfig{}

	if installFlags.PreInstallConfigPath != "" {
		configFilePath = fmt.Sprintf("%s%c%s", basePath, os.PathSeparator, installFlags.PreInstallConfigPath)
	} else {
		// read from configuration
	}

	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		printlnErr("fail to read pre-install config", err)
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, pConf)
	if err != nil {
		printlnErr("fail to unmarshal pre-install config", err)
		return nil, err
	}

	// sort
	sort.Sort(ComparableItems(pConf.Items))

	clientUtils, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, 0)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	for _, item := range pConf.Items {
		fmt.Println(item.Path + " : " + item.Weight)
		files = append(files, basePath+"/"+item.Path)
		// todo check if item.Path is a valid file
		err = clientUtils.Create(basePath+"/"+item.Path, nameSpace, true)
		if err != nil {
			return files, err
		}
	}
	return files, nil
}

func waitUtilReady() error {
	resourceName := ""
	kind := ""
	switch kind {
	case "Job", "Pod": // only wait for job and pod
	default:
		return nil
	}

	selector, err := fields.ParseSelector(fmt.Sprintf("metadata.Name=%s", resourceName))
	if err != nil {
		return err
	}

	restClient, err := GetRestClient()

	lw := cachetools.NewListWatchFromClient(restClient, "Job", nameSpace, selector)
	ctx, cancel := watchtools.ContextWithOptionalTimeout(context.Background(), time.Minute*5)
	defer cancel()
	_, err = watchtools.UntilWithSync(ctx, lw, &unstructured.Unstructured{}, nil, func(e watch.Event) (bool, error) {
		switch e.Type {
		case watch.Added, watch.Modified:
			// For things like a secret or a config map, this is the best indicator
			// we get. We care mostly about jobs, where what we want to see is
			// the status go into a good state. For other types, like ReplicaSet
			// we don't really do anything to support these as hooks.
			switch kind {
			case "Job":
				return waitForJob(e.Object, resourceName)
			case "Pod":
				return waitForPodSuccess(e.Object, resourceName)
			}
			return true, nil
		case watch.Deleted:
			fmt.Printf("Deleted event for %s", resourceName)
			return true, nil
		case watch.Error:
			// Handle error and return with an error.
			fmt.Printf("Error event for %s", resourceName)
			return true, errors.Errorf("failed to deploy %s", resourceName)
		default:
			return false, nil
		}
	})

	return nil
}

func waitForJob(obj runtime.Object, name string) (bool, error) {
	o, ok := obj.(*batch.Job)
	if !ok {
		return true, errors.Errorf("expected %s to be a *batch.Job, got %T", name, obj)
	}

	for _, c := range o.Status.Conditions {
		if c.Type == batch.JobComplete && c.Status == "True" {
			return true, nil
		} else if c.Type == batch.JobFailed && c.Status == "True" {
			return true, errors.Errorf("job failed: %s", c.Reason)
		}
	}

	return false, nil
}

func waitForPodSuccess(obj runtime.Object, name string) (bool, error) {
	o, ok := obj.(*v1.Pod)
	if !ok {
		return true, errors.Errorf("expected %s to be a *v1.Pod, got %T", name, obj)
	}

	switch o.Status.Phase {
	case v1.PodSucceeded:
		fmt.Printf("Pod %s succeeded", o.Name)
		return true, nil
	case v1.PodFailed:
		return true, errors.Errorf("pod %s failed", o.Name)
	case v1.PodPending:
		fmt.Printf("Pod %s pending", o.Name)
	case v1.PodRunning:
		fmt.Printf("Pod %s running", o.Name)
	}

	return false, nil
}
