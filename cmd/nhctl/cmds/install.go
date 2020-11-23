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
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	cachetools "k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"math/rand"
	"nocalhost/internal/nhctl"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/tools"
	"nocalhost/pkg/nhctl/utils"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type InstallFlags struct {
	*EnvSettings
	Url     string // resource url
	AppType string
	//ResourcesDir  string
	HelmValueFile    string
	ForceInstall     bool
	IgnorePreInstall bool
	HelmSet          string
	HelmRepoName     string
	HelmRepoUrl      string
	HelmChartName    string
	Wait             bool
}

var installFlags = InstallFlags{
	EnvSettings: settings,
}

func init() {
	installCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	installCmd.Flags().StringVarP(&installFlags.Url, "url", "u", "", "resource url")
	//installCmd.Flags().StringVarP(&installFlags.ResourcesDir, "dir", "d", "", "the dir of helm package or manifest")
	installCmd.Flags().StringVarP(&installFlags.HelmValueFile, "", "f", "", "helm's Value.yaml")
	installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "", "nocalhostApp type: helm or helm-repo or manifest")
	installCmd.Flags().BoolVar(&installFlags.ForceInstall, "force", installFlags.ForceInstall, "force install")
	installCmd.Flags().BoolVar(&installFlags.Wait, "wait", installFlags.Wait, "wait for completion")
	installCmd.Flags().BoolVar(&installFlags.IgnorePreInstall, "ignore-pre-install", installFlags.IgnorePreInstall, "ignore pre-install")
	installCmd.Flags().StringVar(&installFlags.HelmSet, "set", "", "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	installCmd.Flags().StringVar(&installFlags.HelmRepoName, "helm-repo-name", "", "chart repository name")
	installCmd.Flags().StringVar(&installFlags.HelmRepoUrl, "helm-repo-url", "", "chart repository url where to locate the requested chart")
	installCmd.Flags().StringVar(&installFlags.HelmChartName, "helm-chart-name", "", "chart name")
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
		applicationName := args[0]
		var err error
		if installFlags.Url == "" && installFlags.AppType != string(nhctl.HelmRepo) {
			fmt.Println("error: if app type is not helm-repo , --url must be specified")
			os.Exit(1)
		}
		if installFlags.AppType == string(nhctl.HelmRepo) {
			if installFlags.HelmChartName == "" {
				fmt.Println("error: --helm-chart-name must be specified")
				os.Exit(1)
			}
			if installFlags.HelmRepoUrl == "" && installFlags.HelmRepoName == "" {
				fmt.Println("error: --helm-repo-url or --helm-repo-name must be specified")
				os.Exit(1)
			}
		}
		if nocalhost.CheckIfApplicationExist(applicationName) {
			fmt.Printf("[error] application \"%s\" already exists\n", applicationName)
			os.Exit(1)
		}

		fmt.Println("install application...")
		err = InstallApplication(applicationName)
		if err != nil {
			printlnErr("failed to install application", err)
			debug("clean up resources...")
			err = nocalhost.CleanupAppFiles(applicationName)
			if err != nil {
				fmt.Printf("[error] failed to clean up:%v\n", err)
			} else {
				debug("resources have been clean up")
			}
			os.Exit(1)
		} else {
			fmt.Printf("application \"%s\" is installed", applicationName)
		}
	},
}

func InstallApplication(applicationName string) error {

	var (
		resourcesPath string
		err           error
	)

	// check if kubernetes is available
	client, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, nhctl.DefaultClientGoTimeOut)
	if err != nil {
		printlnErr("kubernetes is unavailable, please check your cluster kubeconfig", err)
		return err
	}

	if nameSpace == "" {
		ns, err := client.GetDefaultNamespace()
		if err != nil {
			fmt.Println("[error] fail to get default namespace, you can use -n to specify a kubernetes namespace")
			return err
		}
		nameSpace = ns
		debug("use default namespace \"%s\"", ns)
	}
	// check if namespace is available
	//timeOutCtx, _ := context.WithTimeout(context.TODO(), nhctl.DefaultClientGoTimeOut)
	//ava, err := client.CheckIfNamespaceIsAccessible(timeOutCtx, nameSpace)
	//if err == nil && ava {
	//	debug("[check] %s is available", nameSpace)
	//} else {
	//	fmt.Printf("[error] \"%s\" is unavailable\n", nameSpace)
	//	return err
	//}

	// create application dir
	applicationDir := GetApplicationHomeDir(applicationName)
	if _, err = os.Stat(applicationDir); err != nil {
		if os.IsNotExist(err) {
			debug("%s not exists, create application dir", applicationDir)
			err = os.Mkdir(applicationDir, 0755)
			if err != nil {
				return err
			}
			//utils.Mush(DownloadApplicationToNhctlHome(applicationDir))
		} else {
			return err
		}
	} else if !installFlags.ForceInstall {
		fmt.Printf("application %s already exists, please use --force to force it to be reinstalled\n", applicationName)
		return errors.New("application already exists, please use --force to force it to be reinstalled")
	} else if installFlags.ForceInstall {
		fmt.Printf("force to reinstall %s\n", applicationName)
		err = os.RemoveAll(applicationDir)
		if err == nil {
			err = os.Mkdir(applicationDir, 0755)
		}
		if err != nil {
			return err
		}
		//utils.Mush(DownloadApplicationToNhctlHome(applicationDir))
	}

	// init application dir
	if installFlags.Url != "" {
		err = DownloadApplicationToNhctlHome(applicationDir)
		if err != nil {
			fmt.Printf("[error] failed to clone : %s\n", installFlags.Url)
			return err
		}
	}

	nocalhostApp, err = nhctl.NewApplication(applicationName)
	if err != nil {
		return err
	}
	//config := nocalhostApp.Config

	if installFlags.AppType != string(nhctl.HelmRepo) {
		resourcesPath = nocalhostApp.GetResourceDir()

		appDep := nocalhostApp.GetDependencies()
		if appDep != nil {
			debug("install dependency config map")
			var depForYaml = &struct {
				Dependency []*nhctl.SvcDependency `json:"dependency" yaml:"dependency"`
			}{
				Dependency: appDep,
			}

			yamlBytes, err := yaml.Marshal(depForYaml)
			if err != nil {
				return err
			}

			dataMap := make(map[string]string, 0)
			dataMap["nocalhost"] = string(yamlBytes)

			configMap := &v1.ConfigMap{
				Data: dataMap,
			}
			//fmt.Println("config map : " + string(yamlBytes))

			var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
			rand.Seed(time.Now().UnixNano())
			b := make([]rune, 4)
			for i := range b {
				b[i] = letterRunes[rand.Intn(len(letterRunes))]
			}
			generateName := fmt.Sprintf("nocalhost-depends-do-not-overwrite-%s", string(b))
			configMap.Name = generateName
			_, err = client.ClientSet.CoreV1().ConfigMaps(nameSpace).Create(context.TODO(), configMap, metav1.CreateOptions{})
			if err != nil {
				fmt.Printf("[error] fail to create dependency config %s, err: %v\n", configMap.Name, err)
				return err
			} else {
				debug("config map %s has been installed, record it", configMap.Name)
				nocalhostApp.AppProfile.DependencyConfigMapName = configMap.Name
				nocalhostApp.SaveProfile()
			}
		}
	} else {
		debug("no dependency config map defined")
	}

	// save application info
	nocalhostApp.AppProfile.Namespace = nameSpace
	nocalhostApp.AppProfile.Kubeconfig = settings.KubeConfig
	err = nocalhostApp.SaveProfile()
	if err != nil {
		fmt.Println("[error] fail to save nocalhostApp profile")
	}

	debug("resources path is %s\n", resourcesPath)
	appType, err := nocalhostApp.GetType()
	if err != nil && installFlags.AppType == "" {
		return err
	}
	if installFlags.AppType != "" {
		appType = nhctl.AppType(installFlags.AppType)
	}
	debug("[nocalhost config] nocalhostApp type: %s", appType)
	switch appType {
	case nhctl.Helm:
		commonParams := make([]string, 0)
		if nameSpace != "" {
			commonParams = append(commonParams, "-n", nameSpace)
		}
		if settings.KubeConfig != "" {
			commonParams = append(commonParams, "--kubeconfig", settings.KubeConfig)
		}
		if settings.Debug {
			commonParams = append(commonParams, "--debug")
		}

		params := []string{"install", applicationName, resourcesPath}
		if installFlags.Wait {
			params = append(params, "--wait")
		}
		if installFlags.HelmSet != "" {
			params = append(params, "--set", installFlags.HelmSet)
		}
		if installFlags.HelmValueFile != "" {
			params = append(params, "-f", installFlags.HelmSet)
		}
		params = append(params, commonParams...)

		fmt.Println("install helm application, this may take several minutes, please waiting...")

		depParams := []string{"dependency", "build", resourcesPath}
		depParams = append(depParams, commonParams...)
		depBuildOutput, err := tools.ExecCommand(nil, false, "helm", depParams...)
		if err != nil {
			printlnErr("fail to build dependency for helm app", err)
			return err
		}
		debug(depBuildOutput)

		output, err := tools.ExecCommand(nil, false, "helm", params...)
		if err != nil {
			printlnErr("fail to install helm nocalhostApp", err)
			return err
		}
		debug(output)
		fmt.Printf(`helm nocalhostApp installed, use "helm list -n %s" to get the information of the helm release`+"\n", nameSpace)
	case nhctl.Manifest:
		excludeFiles := make([]string, 0)
		if nocalhostApp.Config.PreInstall != nil && !installFlags.IgnorePreInstall {
			debug("[nocalhost config] reading pre-install hook")
			excludeFiles, err = PreInstall(resourcesPath, nocalhostApp.Config.PreInstall)
			utils.Mush(err)
		}

		// install manifest recursively, don't install pre-install workload again
		err = InstallManifestRecursively(resourcesPath, excludeFiles)
	case nhctl.HelmRepo:
		commonParams := make([]string, 0)
		if nameSpace != "" {
			commonParams = append(commonParams, "--namespace", nameSpace)
		}
		if settings.KubeConfig != "" {
			commonParams = append(commonParams, "--kubeconfig", settings.KubeConfig)
		}
		if settings.Debug {
			commonParams = append(commonParams, "--debug")
		}

		chartName := installFlags.HelmChartName
		installParams := []string{"install", applicationName}
		if installFlags.Wait {
			installParams = append(installParams, "--wait")
		}
		//if installFlags.HelmRepoUrl
		if installFlags.HelmRepoName != "" {
			installParams = append(installParams, fmt.Sprintf("%s/%s", installFlags.HelmRepoName, chartName))
		} else if installFlags.HelmRepoUrl != "" {
			installParams = append(installParams, chartName, "--repo", installFlags.HelmRepoUrl)
		}
		if installFlags.HelmSet != "" {
			installParams = append(installParams, "--set", installFlags.HelmSet)
		}
		if installFlags.HelmValueFile != "" {
			installParams = append(installParams, "-f", installFlags.HelmSet)
		}
		installParams = append(installParams, commonParams...)

		fmt.Println("install helm application, this may take several minutes, please waiting...")

		//depParams := []string{"dependency", "build", resourcesPath}
		//depParams = append(depParams, commonParams...)
		//depBuildOutput, err := tools.ExecCommand(nil, false, "helm", depParams...)
		//if err != nil {
		//	printlnErr("fail to build dependency for helm app", err)
		//	return err
		//}
		//debug(depBuildOutput)

		output, err := tools.ExecCommand(nil, false, "helm", installParams...)
		if err != nil {
			printlnErr("fail to install helm nocalhostApp", err)
			return err
		}
		debug(output)
		fmt.Printf(`helm nocalhost app installed, use "helm list -n %s" to get the information of the helm release`+"\n", nameSpace)

	default:
		fmt.Println("unsupported application type, it mush be helm or helm-repo or manifest")
		return errors.New("unsupported application type, it mush be helm or helm-repo or manifest")
	}

	nocalhostApp.SetAppType(appType)
	err = nocalhostApp.SetInstalledStatus(true)
	if err != nil {
		return errors.New("fail to update \"installed\" status")
	}
	return nil
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
		// clone git to homePath
		_, err = tools.ExecCommand(nil, true, "git", "clone", "--depth", "1", installFlags.Url, homePath)
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

func InstallManifestRecursively(dir string, excludeFiles []string) error {

	files, _, err := GetFilesAndDirs(dir)
	if err != nil {
		return err
	}
	start := time.Now()
	wg := sync.WaitGroup{}

	clientUtil, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, 0)
outer:
	for _, file := range files {
		for _, ex := range excludeFiles {
			if ex == file {
				fmt.Println("ignore file : " + file)
				continue outer
			}
		}

		// parallel
		wg.Add(1)
		go func(fileName string) {
			fmt.Println("create " + fileName)
			clientUtil.Create(fileName, nameSpace, false)
			wg.Done()
		}(file)
	}
	wg.Wait()
	end := time.Now()
	debug("installing takes %f seconds", end.Sub(start).Seconds())
	return err
}

func PreInstall(basePath string, items []*nhctl.PreInstallItem) ([]string, error) {
	fmt.Println("run pre-install....")

	// sort
	sort.Sort(nhctl.ComparableItems(items))

	clientUtils, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, 0)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	for _, item := range items {
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
