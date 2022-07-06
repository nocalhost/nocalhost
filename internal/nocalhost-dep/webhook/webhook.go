/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package webhook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"io/ioutil"
	v1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nocalhost-dep/cm"
	service_account "nocalhost/internal/nocalhost-dep/serviceaccount"
	nocalhost "nocalhost/pkg/nocalhost-dep/go-client"
	"os"
	"strings"
	"sync"
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
var waitImages = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-wait:latest"

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
var lock = sync.Mutex{}
var watchers = map[string]*service_account.ServiceAccountWatcher{}
var cmWatchers = map[string]*cm.CmWatcher{}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = corev1.AddToScheme(runtimeScheme)
	clientset = nocalhost.InitClientSet()
	cachedRestMapper = nocalhost.InitCachedRestMapper()
}

func getCmWatcher(ns string) *cm.CmWatcher {
	lock.Lock()
	defer lock.Unlock()

	if watcher, ok := cmWatchers[ns]; ok {
		return watcher
	}

	watcher := cm.NewCmWatcher(clientset)
	_ = watcher.Prepare(ns)

	go func() {
		watcher.Watch()
	}()

	cmWatchers[ns] = watcher
	return watcher
}

func getWatcher(ns string) *service_account.ServiceAccountWatcher {
	lock.Lock()
	defer lock.Unlock()

	if watcher, ok := watchers[ns]; ok {
		return watcher
	}

	watcher := service_account.NewServiceAccountWatcher(clientset)
	_ = watcher.Prepare(ns)

	go func() {
		watcher.Watch()
	}()

	watchers[ns] = watcher
	return watcher
}

// (https://github.com/kubernetes/kubernetes/issues/57982)
func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume) {
	defaulter.Default(
		&corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: containers,
				Volumes:    volumes,
			},
		},
	)
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
func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {

	if metadata == nil {
		return false
	}

	// If the environment variable MATCH_NAMESPACE exists,
	// only the namespaces in the environment variable will be matched
	e := os.Getenv("MATCH_NAMESPACE")
	if e != "" {
		e = strings.Replace(e, " ", "", -1)
		matchNamespaces := strings.Split(e, ",")
		matchNamespacesMap := make(map[string]struct{})
		for _, ns := range matchNamespaces {
			matchNamespacesMap[ns] = struct{}{}
		}
		if _, ok := matchNamespacesMap[metadata.Namespace]; !ok {
			glog.Infof(
				"Skip mutation %s/%s, it's not in the MATCH_NAMESPACE %v",
				metadata.Namespace, metadata.Name, matchNamespaces,
			)
			return false
		}
	}

	// skip special kubernete system namespaces
	for _, namespace := range ignoredList {
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

	//status := annotations[admissionWebhookAnnotationStatusKey]

	// determine whether to perform mutation based on annotation for the target resource
	var required = true

	//glog.Infof("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)
	return required
}

// create mutation patch for resources
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
func (whsvr *WebhookServer) mutate(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	req := ar.Request

	var (
		objectMeta    *metav1.ObjectMeta
		resourceName  string
		initContainer []corev1.Container
		containers    []corev1.Container
	)
	resourceType := req.Kind.Kind

	var omh ObjectMetaHolder

	// skip delete operation expect resource quota
	// skip all connect operation
	switch string(req.Operation) {
	case "DELETE":
		if resourceType != "ResourceQuota" {
			return &v1.AdmissionResponse{
				Allowed: true,
			}
		}
	case "CONNECT":
		return &v1.AdmissionResponse{
			Allowed: true,
		}
	default:
		if err := json.Unmarshal(req.Object.Raw, &omh); err != nil {
			glog.Errorf(
				"Could not unmarshal raw object: %v, resource: %+v, name: %s, ns: %s, oper: %+v", err, req.Resource,
				req.Name, req.Namespace, req.Operation,
			)
			return &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		// determine whether to perform mutation
		if !mutationRequired(ignoredNamespaces, &omh.ObjectMeta) {
			glog.Infof("Skipping mutation for %s/%s due to policy check", req.Namespace, req.Name)
			return &v1.AdmissionResponse{
				Allowed: true,
			}
		}
	}

	annotationPair := make(chan []string, 1)
	go func() {
		if &omh == nil {
			annotationPair <- []string{}
		} else {
			ap := omh.getOwnRefSignedAnnotation(req.Namespace)
			annotationPair <- ap
			if len(ap) > 0 {
				glog.Infof(
					"Kind: `%s` Name: `%s` in ns `%s` should patching his signed anno: [%s]", req.Kind,
					req.Name, req.Namespace, strings.Join(ap, ", "),
				)
			}
		}
	}()

	// overwrite nocalhostNamespace for get dep config from devs namespace
	nocalhostNamespace = req.Namespace
	// admission webhook Specific 6 resource blocking
	switch resourceType {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = deployment.Name,
			&deployment.ObjectMeta, deployment.Spec.Template.Spec.InitContainers,
			deployment.Spec.Template.Spec.Containers
	case "DaemonSet":
		var daemonSet appsv1.DaemonSet
		if err := json.Unmarshal(req.Object.Raw, &daemonSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = daemonSet.Name,
			&daemonSet.ObjectMeta, daemonSet.Spec.Template.Spec.InitContainers,
			daemonSet.Spec.Template.Spec.Containers
	case "ReplicaSet":
		var replicaSet appsv1.ReplicaSet
		if err := json.Unmarshal(req.Object.Raw, &replicaSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = replicaSet.Name,
			&replicaSet.ObjectMeta, replicaSet.Spec.Template.Spec.InitContainers,
			replicaSet.Spec.Template.Spec.Containers
	case "StatefulSet":
		var statefulSet appsv1.StatefulSet
		if err := json.Unmarshal(req.Object.Raw, &statefulSet); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = statefulSet.Name,
			&statefulSet.ObjectMeta, statefulSet.Spec.Template.Spec.InitContainers,
			statefulSet.Spec.Template.Spec.Containers
	case "Job":
		var job batchv1.Job
		if err := json.Unmarshal(req.Object.Raw, &job); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = job.Name, &job.ObjectMeta,
			job.Spec.Template.Spec.InitContainers, job.Spec.Template.Spec.Containers
	case "CronJob":
		var cronJob batchv1beta1.CronJob
		if err := json.Unmarshal(req.Object.Raw, &cronJob); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, objectMeta, initContainer, containers = cronJob.Name, &cronJob.ObjectMeta,
			cronJob.Spec.JobTemplate.Spec.Template.Spec.InitContainers,
			cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers
	case "ResourceQuota":
		namespace := req.Namespace

		glog.Infof(
			fmt.Sprintf(
				"request for ns %s, req.Name %s, reqName %s, uuid %s", namespace, omh.Name, req.Name,
				req.UserInfo.UID,
			),
		)
		glog.Infof(fmt.Sprintf("request for %s", omh.String()))

		if req.UserInfo.UID == "" || (omh.Name != "rq-"+namespace && req.Name != "rq-"+namespace) {
			return &v1.AdmissionResponse{
				Allowed: true,
			}
		}

		isClusterAdmin := getWatcher(namespace).IsClusterAdmin(req.UserInfo.UID)

		// try load sa from default namespace
		if isClusterAdmin == nil {
			isClusterAdmin = getWatcher("default").IsClusterAdmin(req.UserInfo.UID)
		}

		if isClusterAdmin == nil {
			glog.Infof(
				fmt.Sprintf(
					"request for ns %s, resourcename %s, reqName %s, isClusteradmin is null, uid %s",
					namespace, resourceName, req.Name, req.UserInfo.UID,
				),
			)
			marshal, _ := json.Marshal(req)

			var err = fmt.Errorf("Could not get service account with uuid: %s, %s", req.UserInfo.UID, marshal)
			glog.Error(err)
			return &v1.AdmissionResponse{
				Allowed: true,
			}
		}

		if *isClusterAdmin {
			return &v1.AdmissionResponse{
				Allowed: true,
			}
		}

		glog.Infof(
			"Request user uuid %s, Sa uuid %s without cluster-admin role, so resource quota request denied",
			req.UserInfo.UID, req.UserInfo.UID,
		)
		return &v1.AdmissionResponse{
			Allowed: false,
		}
	}

	if resourceType != "ResourceQuota" {
		glog.Infof(
			"AdmissionReview for Kind=%v, Namespace=%v Name=%v ReqName=%v UID=%v patchOperation=%v UserInfo=%v",
			req.Kind, req.Namespace, resourceName, req.Name, req.UID, req.Operation, req.UserInfo,
		)
	}

	var appName string

	// getting application name from owner ref
	ap := <-annotationPair
	if len(ap) == 2 {
		appName = ap[1]
	} else {
		appName = _const.DefaultNocalhostApplication
		ap = []string{_const.NocalhostApplicationName, _const.DefaultNocalhostApplication}
	}

	var injectInitContainers []corev1.Container
	var EnvVar []envVar
	var err error
	var configSuccessfulLoaded = false

	// caution, should use resourceName instead of req.Name to inject initContainer and env
	// or inject container into configmap、service or something, it's unexpected!!

	// inject nocalhost config from
	// 1. annotations (dev.nocalhost)
	// 2. cm (dev.nocalhost.config.${appName})
	// 3. nocalhost-dep-map
	if v, ok := omh.Annotations[appmeta.AnnotationKey]; ok && v != "" && resourceName != "" {
		injectInitContainers, EnvVar, err = nocalhostDepConfigmapCustom(
			func() (*profile.NocalHostAppConfigV2, *profile.ServiceConfigV2, error) {
				return app.LoadSvcCfgFromStrIfValid(v, resourceName, base.SvcType(resourceType))
			}, containers,
		)

		if err != nil {
			glog.Infof(
				"Admission Config Resolve Err from annotation for Kind=%v, Namespace=%v, Name=%v, Error: %v",
				req.Kind, req.Namespace, resourceName, err,
			)
			err = nil
		} else {
			configSuccessfulLoaded = true
			glog.Infof(
				"Admission Config Success Resolve from annotation for Kind=%v, Namespace=%v Name=%v",
				req.Kind, req.Namespace, resourceName,
			)
		}
	}

	if !configSuccessfulLoaded && resourceName != "" {
		if appCfg, svcCfg, err := getCmWatcher(req.Namespace).GetNocalhostConfig(
			appName, resourceType, resourceName,
		); err != nil {
			if err != cm.NOT_FOUND {
				glog.Infof(
					"Admission Config Resolve Err from configmap for Kind=%v, Namespace=%v, Name=%v, Error: %v",
					req.Kind, req.Namespace, resourceName, err,
				)
			}
			err = nil
		} else {

			if injectInitContainers, EnvVar, err = nocalhostDepConfigmapCustom(
				func() (*profile.NocalHostAppConfigV2, *profile.ServiceConfigV2, error) {
					return appCfg, svcCfg, nil
				}, containers,
			); err != nil {
				glog.Infof(
					"Admission Config Resolve Err from configmap while load svc for Kind=%v, Namespace=%v, Name=%v, Error: %v",
					req.Kind, req.Namespace, resourceName, err,
				)
				err = nil
			} else {
				configSuccessfulLoaded = true
				glog.Infof(
					"Admission Config Success Resolve from configmap for Kind=%v, Namespace=%v Name=%v",
					req.Kind, req.Namespace, resourceName,
				)
			}
		}
	}

	if !configSuccessfulLoaded {
		// configmap
		if injectInitContainers, EnvVar, err = nocalhostDepConfigmap(
			nocalhostNamespace, resourceName, resourceType, objectMeta, containers,
		); err != nil {
			glog.Infof(
				"Admission Config Resolve Err for Kind=%v, Namespace=%v, Name=%v, Error: %v",
				req.Kind, req.Namespace, resourceName, err,
			)
			err = nil
		} else {
			configSuccessfulLoaded = true
			glog.Infof(
				"Admission Config Success Resolve from nocalhostDepConfigmap for Kind=%v, Namespace=%v, Name=%v",
				req.Kind, req.Namespace, resourceName,
			)
		}
	}

	// Workaround: https://github.com/kubernetes/kubernetes/issues/57982
	applyDefaultsWorkaround(whsvr.SidecarConfig.Containers, whsvr.SidecarConfig.Volumes)

	p := Patcher{}
	p.patchAnnotations(omh.Annotations, ap)
	p.patchInitContainer(initContainer, injectInitContainers)
	p.patchEnv(containers, EnvVar)
	patchBytes, err := p.patchBytes()

	glog.Infof("patchBytes %s", string(patchBytes))
	if err != nil {
		return &v1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &v1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1.PatchType {
			pt := v1.PatchTypeJSONPatch
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

	var admissionResponse *v1.AdmissionResponse
	ar := v1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = whsvr.mutate(&ar)
	}

	admissionReview := v1.AdmissionReview{}
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
