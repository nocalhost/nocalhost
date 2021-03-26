package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"strings"
	"time"
)

type Patcher struct {
	patch []patchOperation
}

func (p *Patcher) patchInitContainer(objectInitContainer []corev1.Container, initContainers []corev1.Container) {
	if initContainers != nil && len(initContainers) > 0 {
		p.patch = append(p.patch, addInitContainer(objectInitContainer, initContainers, "/spec/template/spec/initContainers")...)
	}
}

func (p *Patcher) patchEnv(objContainers []corev1.Container, envVar []envVar) {
	if envVar != nil && len(envVar) > 0 {
		for k, v := range objContainers {
			p.patch = append(p.patch, addContainerEnvVar(k, v.Env, envVar)...)
		}
	}
}

func (p *Patcher) patchAnnotations(currentAnnos map[string]string, kvPair []string) {
	if len(kvPair) > 0 {
		if len(currentAnnos) == 0 {
			currentAnnos = map[string]string{}
		}

		currentAnnos[kvPair[0]] = kvPair[1]
		p.patch = append(p.patch, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: currentAnnos,
		})
	}
}

func (p *Patcher) patchBytes() ([]byte, error) {
	fmt.Printf("patch %+v\n", p.patch)
	return json.Marshal(p.patch)
}

func addContainerEnvVar(k int, target []corev1.EnvVar, envVar []envVar) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range envVar {
		for _, env := range add.EnvVar {
			if add.ContainerIndex != k {
				continue
			}
			value = env
			path := "/spec/template/spec/containers/" + strconv.Itoa(add.ContainerIndex) + "/env"
			if first {
				first = false
				value = []corev1.EnvVar{env}
			} else {
				path = path + "/-"
			}
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  path,
				Value: value,
			})
		}
	}
	return patch
}

// get nocalhost dependents configmaps, this will get from specify namespace by labels
// nhctl will create dependency configmap in users dev space
func nocalhostDepConfigmap(namespace string, resourceName string, resourceType string, objectMeta *metav1.ObjectMeta, containers []corev1.Container) ([]corev1.Container, []envVar, error) {
	// labelSelector="use-for=nocalhost-dep"
	labelSelector := map[string]string{
		"use-for": "nocalhost-dep",
	}
	setLabelSelector := labels.Set(labelSelector)
	startTime := time.Now()
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: setLabelSelector.AsSelector().String()})
	duration := time.Now().Sub(startTime)
	glog.Infof("get configmap total cost %d", duration.Milliseconds())
	initContainers := make([]corev1.Container, 0)
	envVarArray := make([]envVar, 0)
	if err != nil {
		glog.Fatalln("failed to get config map:", err)
		return initContainers, envVarArray, err
	}
	for i, cm := range configMaps.Items {
		fmt.Printf("[%d] %s\n", i, cm.GetName())
		if strings.Contains(cm.GetName(), "nocalhost-depends-do-not-overwrite") { // Dependency description configmap
			if configMapValue, ok := cm.Data["nocalhost"]; ok {
				fmt.Printf("[%d] %s\n", i, configMapValue)
				dep := mainDep{}
				err := yaml.Unmarshal([]byte(configMapValue), &dep)
				if err != nil {
					glog.Fatalf("failed to unmarshal configmap: %s\n", cm.GetName())
				}
				fmt.Printf("%+v\n", dep)
				// inject install global env
				for _, env := range dep.Env.Global {
					for k := range containers {
						addEnvList := make([]corev1.EnvVar, 0)
						addEnv := corev1.EnvVar{
							Name:  env.Name,
							Value: env.Value,
						}
						addEnvList = append(addEnvList, addEnv)
						envVarEach := envVar{
							EnvVar:         addEnvList,
							ContainerIndex: k,
						}
						envVarArray = append(envVarArray, envVarEach)
					}
				}
				// inject install service env
				for _, env := range dep.Env.Service {
					if env.Name == resourceName && (strings.ToLower(env.Type) == strings.ToLower(resourceType) || dep.ReleaseName+"-"+env.Name == resourceName) {
						for _, container := range env.Container {
							for k, objContainer := range containers {
								addEnvList := make([]corev1.EnvVar, 0)
								// match name or match all
								if container.Name == objContainer.Name || container.Name == "" {
									for _, envFromConfig := range container.InstallEnv {
										addEnv := corev1.EnvVar{
											Name:  envFromConfig.Name,
											Value: envFromConfig.Value,
										}
										addEnvList = append(addEnvList, addEnv)
									}
									envVarEach := envVar{
										EnvVar:         addEnvList,
										ContainerIndex: k,
									}
									envVarArray = append(envVarArray, envVarEach)
								}
							}
						}
					}
				}

				for key, dependency := range dep.Dependency {
					// K8S native type is case-sensitive, dependent descriptions are not distinguished, and unified into lowercase
					// if has metadata.labels.release, then release-name should fix as dependency.Name
					// helm install my-pro prometheus, deployment will be set my-pro-prometheus-alertmanager, if dependency set prometheus-alertmanager it will regrade as resourceName
					if dependency.Name == resourceName && (strings.ToLower(dependency.Type) == strings.ToLower(resourceType) || dep.ReleaseName+"-"+dependency.Name == resourceName) {
						// initContainer
						if dependency.Pods != nil {
							args := func(podsList []string) []string {
								var args []string
								// args = append(args, "sh", "-c")
								for key, pod := range podsList {
									if key != 0 {
										args = append(args, "&&")
									}
									args = append(args, "wait_for.sh", "pod")
									if strings.ContainsAny(pod, "=") { // means define label, such as app.kubernetes.io/name=nginx
										args = append(args, fmt.Sprintf("-l%s", pod))
									} else { // has not define label, default app label
										args = append(args, fmt.Sprintf("-lapp=%s", pod))
									}
								}
								return args
							}(dependency.Pods)

							waitCmd := strings.Join(args, " ")
							var cmd []string
							cmd = append(cmd, "sh", "-c", waitCmd)

							initContainer := corev1.Container{
								Name:            "wait-for-pods-" + strconv.Itoa(i) + strconv.Itoa(key),
								Image:           waitImages,
								ImagePullPolicy: corev1.PullPolicy("Always"),
								Command:         cmd,
							}
							initContainers = append(initContainers, initContainer)
						}
						if dependency.Jobs != nil {
							args := func(jobsList []string) []string {
								var args []string
								// args = append(args, "sh", "-c")
								for key, job := range jobsList {
									if key != 0 {
										args = append(args, "&&")
									}
									args = append(args, "wait_for.sh", "job")
									if strings.ContainsAny(job, "=") { // means define label, such as app.kubernetes.io/name=nginx
										args = append(args, fmt.Sprintf("-l%s", job))
									} else { // has not define label, default app label
										args = append(args, fmt.Sprintf("-lapp=%s", job))
									}
								}
								return args
							}(dependency.Jobs)

							waitCmd := strings.Join(args, " ")
							var cmd []string
							cmd = append(cmd, "sh", "-c", waitCmd)

							initContainer := corev1.Container{
								Name:            "wait-for-jobs-" + strconv.Itoa(i) + strconv.Itoa(key),
								Image:           waitImages,
								ImagePullPolicy: corev1.PullPolicy("Always"),
								Command:         cmd,
							}
							initContainers = append(initContainers, initContainer)
						}
					}
				}
			}
		}
	}
	return initContainers, envVarArray, err
}

// add initContainers
func addInitContainer(objectMeta []corev1.Container, initContainers []corev1.Container, path string) (patch []patchOperation) {
	first := len(objectMeta) == 0
	var value interface{}
	for _, add := range initContainers {
		value = add
		path := path
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}
