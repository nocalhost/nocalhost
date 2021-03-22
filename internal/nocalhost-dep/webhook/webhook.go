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

package webhook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"net/http"
	nocalhost "nocalhost/pkg/nocalhost-dep/go-client"
	"strings"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

var nocalhostNamespace = "nocalhost-reserved"
var waitImages = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-wait:latest"
var imagePullPolicy = "Always"

const (
	admissionWebhookAnnotationInjectKey = "sidecar-injector-webhook.nocalhost/inject"
	admissionWebhookAnnotationStatusKey = "sidecar-injector-webhook.nocalhost/status"
)

type WebhookServer struct {
	SidecarConfig *Config
	Server        *http.Server
}

// Webhook Server parameters
type WhSvrParameters struct {
	Port           int    // webhook server port
	CertFile       string // path to the x509 certificate for https
	KeyFile        string // path to the x509 private key matching `CertFile`
	SidecarCfgFile string // path to sidecar injector configuration file
}

type Config struct {
	Containers []corev1.Container `yaml:"containers"`
	Volumes    []corev1.Volume    `yaml:"volumes"`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type envVar struct {
	EnvVar         []corev1.EnvVar
	ContainerIndex int
}

// dep struct
type mainDep struct {
	Env         globalEnv `yaml:"env"`
	Dependency  []depApp
	ReleaseName string `yaml:"releaseName"`
}

type globalEnv struct {
	Global  []installEnv `yaml:"global"`
	Service []envList    `yaml:"service"`
}

type envList struct {
	Name      string
	Type      string
	Container []containerList
}

type containerList struct {
	Name       string
	InstallEnv []installEnv `yaml:"installEnv"`
}

type installEnv struct {
	Name  string
	Value string
}

type depApp struct {
	Name string
	Type string
	Pods []string
	Jobs []string
}

//type depType struct {
//	Type string `yaml:"type"`
//}
//
//type depPods struct {
//	Pods []string `yaml:"pods"`
//}

type depJobs struct {
	Jobs []string `yaml:"jobs"`
}

var clientset *kubernetes.Clientset
var cachedRestMapper *restmapper.DeferredDiscoveryRESTMapper

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = corev1.AddToScheme(runtimeScheme)
	clientset = nocalhost.InitClientSet()
	cachedRestMapper = nocalhost.InitCachedRestMapper()
}

// (https://github.com/kubernetes/kubernetes/issues/57982)
func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume) {
	defaulter.Default(&corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,
		},
	})
}

func LoadConfig(configFile string) (*Config, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Check whether the target resoured need to be mutated
func mutationRequired(resourceType string, ignoredList []string, metadata *metav1.ObjectMeta) bool {
	switch resourceType {
	case "SubjectAccessReview":
		return false
	case "SelfSubjectAccessReview":
		return false
	case "Event":
		return false
	}

	// skip special kubernete system namespaces
	for _, namespace := range ignoredList {
		if metadata == nil {
			return false
		}

		if metadata.Namespace == namespace {
			glog.Infof("Skip mutation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	// ignore recreate resource for devend
	if annotations["nocalhost-dep-ignore"] == "true" {
		return false
	}

	status := annotations[admissionWebhookAnnotationStatusKey]

	// determine whether to perform mutation based on annotation for the target resource
	var required = true

	glog.Infof("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)
	return required
}

func addContainer(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
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

func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
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

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

// create mutation patch for resoures
//func createPatch(pod *corev1.Pod, sidecarConfig *Config, annotations map[string]string) ([]byte, error) {
//	var patch []patchOperation
//
//	patch = append(patch, addContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)
//	patch = append(patch, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
//	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)
//
//	return json.Marshal(patch)
//}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request

	var (
		objectMeta    *metav1.ObjectMeta
		resourceName  string
		initContainer []corev1.Container
		containers    []corev1.Container
	)
	resourceType := req.Kind.Kind

	var omh ObjectMetaHolder
	if err := json.Unmarshal(req.Object.Raw, &omh); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	annotationPair := make(chan []string, 1)
	go func() {
		ap := omh.getOwnRefSignedAnnotation(req.Namespace)
		annotationPair <- ap
		if len(ap) > 0 {
			glog.Infof("Kind: `%s` Name: `%s` in ns `%s` should patching his signed anno: [%s], meta: %s", req.Kind, req.Name, req.Namespace, strings.Join(ap, ", "), string(req.Object.Raw))
		}
	}()

	// determine whether to perform mutation
	if !mutationRequired(resourceType, ignoredNamespaces, &omh.ObjectMeta) {
		glog.Infof("Skipping mutation for %s/%s due to policy check", req.Namespace, req.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	// overwrite nocalhostNamespace for get dep config from devs namespace
	nocalhostNamespace = req.Namespace
	// admission webhook Specific 6 resource blocking
	switch resourceType {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = deployment.Name, &deployment.ObjectMeta, deployment.Spec.Template.Spec.InitContainers, deployment.Spec.Template.Spec.Containers
	case "DaemonSet":
		var daemonSet appsv1.DaemonSet
		if err := json.Unmarshal(req.Object.Raw, &daemonSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = daemonSet.Name, &daemonSet.ObjectMeta, daemonSet.Spec.Template.Spec.InitContainers, daemonSet.Spec.Template.Spec.Containers
	case "ReplicaSet":
		var replicaSet appsv1.ReplicaSet
		if err := json.Unmarshal(req.Object.Raw, &replicaSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = replicaSet.Name, &replicaSet.ObjectMeta, replicaSet.Spec.Template.Spec.InitContainers, replicaSet.Spec.Template.Spec.Containers
	case "StatefulSet":
		var statefulSet appsv1.StatefulSet
		if err := json.Unmarshal(req.Object.Raw, &statefulSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = statefulSet.Name, &statefulSet.ObjectMeta, statefulSet.Spec.Template.Spec.InitContainers, statefulSet.Spec.Template.Spec.Containers
	case "Job":
		var job batchv1.Job
		if err := json.Unmarshal(req.Object.Raw, &job); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = job.Name, &job.ObjectMeta, job.Spec.Template.Spec.InitContainers, job.Spec.Template.Spec.Containers
	case "CronJob":
		var cronJob batchv1beta1.CronJob
		if err := json.Unmarshal(req.Object.Raw, &cronJob); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = cronJob.Name, &cronJob.ObjectMeta, cronJob.Spec.JobTemplate.Spec.Template.Spec.InitContainers, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers
	case "ResourceQuota":
		//if req.UserInfo.UID == "" {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
		//}

		sa := getSaByUid(req.UserInfo.UID)
		if sa == nil {
			var err = fmt.Errorf("Could not get service account with uuid: %s ", req.UserInfo.UID)
			glog.Error(err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		isAdmin, err := isClusterAdmin(sa)
		if err != nil {
			glog.Errorf("Could not get role-binding from namespace %s, Err: %s", sa.Namespace, err.Error())
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		if isAdmin {
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}

		glog.Infof("Request user uuid %s, Sa uuid %s is from ns %s without cluster-admin role, so resource quota request denied", req.UserInfo.UID, sa.UID, sa.Namespace)
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}

	if resourceType != "ResourceQuota" {
		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
			req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
		glog.Infof("unmarshal for Kind=%v, Namespace=%v Name=%v",
			req.Kind, req.Namespace, req.Name)
	}

	// configmap
	initContainers, EnvVar, err := nocalhostDepConfigmap(nocalhostNamespace, resourceName, resourceType, objectMeta, containers)
	glog.Infof("initContainers %v", initContainers)
	glog.Infof("EnvVar %v", EnvVar)

	// Workaround: https://github.com/kubernetes/kubernetes/issues/57982
	applyDefaultsWorkaround(whsvr.SidecarConfig.Containers, whsvr.SidecarConfig.Volumes)

	ap := <-annotationPair

	p := Patcher{}
	p.patchAnnotations(omh.Annotations, ap)
	p.patchInitContainer(initContainer, initContainers)
	p.patchEnv(containers, EnvVar)
	patchBytes, err := p.patchBytes()

	glog.Infof("patchBytes %s", string(patchBytes))
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("patchBytes %s\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = whsvr.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
