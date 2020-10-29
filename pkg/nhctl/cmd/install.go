package cmd

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
	"sort"
	"strconv"
	"strings"
	"time"
)

var releaseName, gitUrl, resourcesDir, helmValueFile, appType, preInstallConfigPath string

func init() {
	installCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	installCmd.Flags().StringVarP(&releaseName, "release-releaseName", "r", "", "release releaseName of helm")
	installCmd.Flags().StringVarP(&gitUrl, "git-url", "u", "", "url of git")
	installCmd.Flags().StringVarP(&resourcesDir, "dir", "d", "", "the dir of helm package or manifest")
	installCmd.Flags().StringVarP(&helmValueFile, "", "f", "", "helm's Value.yaml")
	installCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubernetes cluster config")
	installCmd.Flags().StringVarP(&appType, "type", "t", "helm", "app type: helm or manifest")
	installCmd.Flags().StringVarP(&preInstallConfigPath, "pre-install", "p", "", "resources to be installed before application install, should be a yaml file path")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install k8s application",
	Long:  `install k8s application`,
	Run: func(cmd *cobra.Command, args []string) {
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		if appType == "helm" && releaseName == "" {
			fmt.Println("error: please use -r to specify the release name of helm")
			return
		}
		//if resourcesDir == "" {
		//	fmt.Println("error: please use -d to specify the dir of helm package")
		//	return
		//}
		if gitUrl == "" {
			fmt.Println("error: please use -u to specify url of git")
			return
		}
		fmt.Println("install application...")
		//if preInstallConfigPath != "" {
		//	PreInstall()
		//}
		InstallApplication()
	},
}

func InstallApplication() {

	// clone git
	_, err := tools.ExecCommand(nil, true, "git", "clone", gitUrl)
	if err != nil {
		printlnErr("fail to clone git", err)
		return
	}

	// helm install
	gitSuffix := gitUrl[:len(gitUrl) - 4]
	fmt.Println("git dir : " + gitSuffix)
	strs := strings.Split(gitSuffix, "/")
	gitSuffix = strs[len(strs)-1]
	resourcesPath := gitSuffix
	if resourcesDir != "" {
		resourcesPath += "/" + resourcesDir
	}
	fmt.Printf("resources path is %s\n", resourcesPath)
	if appType == "helm" {
		_, err = tools.ExecCommand(nil, true, "helm", "upgrade", "--install", "--wait", releaseName, resourcesPath, "-n", nameSpace, "--kubeconfig", kubeconfig)
		if err != nil {
			printlnErr("fail to install helm app", err)
			return
		}
	} else if appType == "manifest" {
		excludeFiles := make([]string, 0)
		if preInstallConfigPath != "" {
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

func InstallManifestRecursively(dir string, excludeFiles []string) error{

	files, _, err := GetFilesAndDirs(dir)
	if err != nil {
		return err
	}

	outer:
	for _, file := range files {
		for _, ex := range excludeFiles{
			if ex == file {
				fmt.Println("ignore file : " + file)
				continue outer
			}
		}
		fmt.Println("create " + file)
		clientUtil, err := clientgoutils.NewClientGoUtils(kubeconfig)
		if err != nil {
			return err
		}
		clientUtil.Create(file,nameSpace,false)
	}
	return err
}

func PreInstall(basePath string) ([]string, error){
	fmt.Println("run pre-install....")
	//  读取一个yaml文件
	pConf := &PreInstallConfig{}
	yamlFile, err := ioutil.ReadFile(preInstallConfigPath)
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

	clientUtils, err := clientgoutils.NewClientGoUtils(kubeconfig)
	if err != nil {
		return nil, err
	}
	files := make([]string,0)
	for _, item := range pConf.Items {
		fmt.Println(item.Path + " : " + item.Weight)
		files = append(files, basePath + "/"+ item.Path)
		// todo check if item.Path is a valid file
		err = clientUtils.Create(basePath + "/" + item.Path, nameSpace, true)
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

	selector, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", resourceName))
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
