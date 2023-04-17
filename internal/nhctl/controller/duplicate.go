/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
)

const (
	IdentifierKey         = "identifier"
	OriginWorkloadNameKey = "origin-workload-name"
	OriginWorkloadTypeKey = "origin-workload-type"

	EnvoySidecarImage         = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-envoy:v1"
	MeshControlPlaneImage     = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-control-plane:v1"
	MeshManager               = "mesh-manager"
	AnnotationMeshEnable      = "dev.mesh.nocalhost.dev"
	AnnotationMeshUuid        = "dev.mesh.nocalhost.dev/uuid"
	AnnotationMeshType        = "dev.mesh.nocalhost.dev/type"
	AnnotationMeshHeaderKey   = "dev.mesh.nocalhost.dev/header-key"
	AnnotationMeshHeaderValue = "dev.mesh.nocalhost.dev/header-value"
	AnnotationMeshPort        = "dev.mesh.nocalhost.dev/port"
	AnnotationMeshTypeDev     = "dev"
	AnnotationMeshTypeOrigin  = "origin"
	EnvoyMeshSidecarName      = "nocalhost-mesh"
)

// ReplaceDuplicateModeImage Create a duplicate deployment instead of replacing image
func (c *Controller) ReplaceDuplicateModeImage(ctx context.Context, ops *model.DevStartOptions) error {
	c.Client.Context(ctx)

	um, err := c.GetUnstructured()
	umClone := um.DeepCopy()
	if err != nil {
		return err
	}

	if c.IsInReplaceDevMode() {
		od, err := GetAnnotationFromUnstructured(um, _const.OriginWorkloadDefinition)
		if err != nil {
			return err
		}

		if um, err = c.Client.GetUnstructuredFromString(od); err != nil {
			return err
		}
	}

	RemoveUselessInfo(um)

	var podTemplate *v1.PodTemplateSpec
	var podTemplateOrigin *v1.PodTemplateSpec
	if !c.DevModeAction.Create {

		um.SetName(c.getDuplicateResourceName())
		um.SetLabels(c.getDuplicateLabelsMap())
		um.SetResourceVersion("")

		if podTemplate, err = GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object); err != nil {
			return err
		}
		podTemplateOrigin, err = GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, umClone.Object)
		if err != nil {
			return err
		}

		podTemplate.Labels = c.getDuplicateLabelsMap()
		podTemplate.Annotations = c.getDevContainerAnnotations(ops.Container, podTemplate.Annotations)

		devContainer, sideCarContainer, devModeVolumes, err :=
			c.genContainersAndVolumes(&podTemplate.Spec, ops.Container, ops.DevImage, ops.StorageClass, true)
		if err != nil {
			return err
		}

		patchDevContainerToPodSpec(&podTemplate.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)
		// add envoy sidecar
		if len(ops.MeshHeader) != 0 {
			err = createMeshManagerIfNotExist(ctx, c.Client.ClientSet, c.NameSpace)
			if err != nil {
				return err
			}
			uuid := podTemplate.GetAnnotations()[AnnotationMeshUuid]
			if len(uuid) == 0 {
				uuid = string(umClone.GetUID())
			}
			addAnnotationToDuplicate(podTemplate, uuid, ops.MeshHeader)
			AddEnvoySidecarForMesh(podTemplate)
			if exist := AddEnvoySidecarForMesh(podTemplateOrigin); !exist {
				addAnnotationToMesh(podTemplateOrigin, uuid)
				// update origin workloads
				err = patchOriginWorkloads(umClone, podTemplateOrigin, c.DevModeAction.PodTemplatePath, c.Client)
				if err != nil {
					return err
				}
			}
		}

		jsonObj, err := um.MarshalJSON()
		if err != nil {
			return errors.WithStack(err)
		}

		lm := map[string]interface{}{"matchLabels": c.getDuplicateLabelsMap()}
		lmJson, _ := json.Marshal(lm)

		var jsonStr string
		pathItems := strings.Split(c.DevModeAction.PodTemplatePath, "/")
		pis := make([]string, 0)
		for _, item := range pathItems {
			if item != "" {
				pis = append(pis, item)
			}
		}
		path := strings.Join(pis[:len(pis)-1], ".")
		selectorPath := path + "." + "selector" // /spec/selector
		if jsonStr, err = sjson.SetRaw(string(jsonObj), selectorPath, string(lmJson)); err != nil {
			return errors.WithStack(err)
		}

		if jsonStr, err = sjson.Delete(jsonStr, "status"); err != nil {
			return errors.WithStack(err)
		}

		pss, _ := json.Marshal(podTemplate)
		templatePath := strings.Join(pis, ".")
		if jsonStr, err = sjson.SetRaw(jsonStr, templatePath, string(pss)); err != nil {
			return errors.WithStack(err)
		}

		infos, err := c.Client.GetResourceInfoFromString(jsonStr, true)
		if err != nil {
			return err
		}

		if len(infos) != 1 {
			return errors.New(fmt.Sprintf("ResourceInfo' num is %d(not 1?)", len(infos)))
		}

		log.Infof("Creating %s(%v)", infos[0].Name, infos[0].Object.GetObjectKind().GroupVersionKind())
		err = c.Client.ApplyResourceInfo(infos[0], nil)
		if err != nil {
			return err
		}

		gvk := infos[0].Object.GetObjectKind().GroupVersionKind()
		kind := gvk.Kind
		if gvk.Version != "" {
			kind += "." + gvk.Version
		}
		if gvk.Group != "" {
			kind += "." + gvk.Group
		}

		for _, item := range c.DevModeAction.ScalePatches {
			log.Infof("Patching %s", item.Patch)
			if err = c.Client.Patch(kind, infos[0].Name, item.Patch, item.Type); err != nil {
				return err
			}
		}

		c.patchAfterDevContainerReplaced(ops.Container, kind, infos[0].Name)
	} else {
		labelsMap := c.getDuplicateLabelsMap()

		if podTemplate, err = GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object); err != nil {
			return err
		}
		podTemplate.Annotations = c.getDevContainerAnnotations(ops.Container, podTemplate.Annotations)
		genDeploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   c.getDuplicateResourceName(),
				Labels: labelsMap,
			},
			Spec: appsv1.DeploymentSpec{
				Template: *podTemplate,
			},
		}
		genDeploy.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
		genDeploy.Spec.Template.Labels = labelsMap
		genDeploy.ResourceVersion = ""
		genDeploy.Spec.Template.Spec.NodeName = ""

		devContainer, sideCarContainer, devModeVolumes, err :=
			c.genContainersAndVolumes(
				&genDeploy.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, true,
			)
		if err != nil {
			return err
		}

		patchDevContainerToPodSpec(
			&genDeploy.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes,
		)

		genDeploy.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyAlways

		podTemplate = &genDeploy.Spec.Template

		// Create generated deployment
		if _, err = c.Client.CreateDeploymentAndWait(genDeploy); err != nil {
			return err
		}

		c.patchAfterDevContainerReplaced(ops.Container, genDeploy.Kind, genDeploy.Name)
	}

	delete(podTemplate.Labels, "pod-template-hash")
	c.devModePodLabels = podTemplate.Labels

	c.waitDevPodToBeReady()
	return nil
}

func addAnnotationToDuplicate(podTemplate *v1.PodTemplateSpec, uuid string, header map[string]string) {
	var k, v string
	for k, v = range header {
		break
	}
	anno := podTemplate.GetAnnotations()
	if anno == nil {
		anno = map[string]string{}
	}
	anno[AnnotationMeshUuid] = uuid
	anno[AnnotationMeshEnable] = "true"
	anno[AnnotationMeshType] = AnnotationMeshTypeDev
	anno[AnnotationMeshPort] = strconv.Itoa(int(podTemplate.Spec.Containers[0].Ports[0].ContainerPort))
	anno[AnnotationMeshHeaderKey] = k
	anno[AnnotationMeshHeaderValue] = v
	podTemplate.SetAnnotations(anno)
}

func addAnnotationToMesh(podTemplate *v1.PodTemplateSpec, uuid string) {
	anno := podTemplate.GetAnnotations()
	if anno == nil {
		anno = map[string]string{}
	}
	anno[AnnotationMeshUuid] = uuid
	anno[AnnotationMeshEnable] = "true"
	anno[AnnotationMeshType] = AnnotationMeshTypeOrigin
	podTemplate.SetAnnotations(anno)
}

func AddEnvoySidecarForMesh(spec *v1.PodTemplateSpec) (exist bool) {
	for i := 0; i < len(spec.Spec.Containers); i++ {
		if spec.Spec.Containers[i].Name == EnvoyMeshSidecarName {
			exist = true
			spec.Spec.Containers = append(spec.Spec.Containers[:i], spec.Spec.Containers[i+1:]...)
			i--
		}
	}
	var port = sets.NewString()
	for _, container := range spec.Spec.Containers {
		for _, containerPort := range container.Ports {
			port.Insert(strconv.Itoa(int(containerPort.ContainerPort)))
		}
	}
	if len(port) == 0 {
		port.Insert("8080", "80")
	}
	t := true
	spec.Spec.Containers = append(spec.Spec.Containers, v1.Container{
		Name:  EnvoyMeshSidecarName,
		Image: EnvoySidecarImage,
		Args: []string{
			"envoy",
			"-c",
			"/etc/envoy/envoy.yaml",
			"-l",
			"trace",
			"--service-node",
			"$(POD_NAME)",
		},
		Env: []v1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
			{
				Name:  "NOCALHOST_PORT",
				Value: strings.Join(port.List(), ","),
			}},
		// TODO: get image pull policy from config
		ImagePullPolicy: v1.PullIfNotPresent,
		SecurityContext: &v1.SecurityContext{
			Privileged: &t,
		},
	})
	return
}

// create mesh-manager if needed, resources: role, serviceAccount, roleBinding, deployment, service
func createMeshManagerIfNotExist(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	_, err := clientset.AppsV1().Deployments(namespace).Get(ctx, MeshManager, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	// already exist, do nothing
	if err == nil {
		return nil
	}

	// create role if not exists
	_, err = clientset.RbacV1().Roles(namespace).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MeshManager,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{"list", "watch", "get"},
			APIGroups: []string{""},
			Resources: []string{"pods"},
		}},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	// create service account if not exists
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MeshManager,
			Namespace: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	// create roleBinding if not exists
	_, err = clientset.RbacV1().RoleBindings(namespace).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MeshManager,
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      MeshManager,
			Namespace: namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     MeshManager,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	// create mesh-manager if not exists
	m := map[string]string{"app": "mesh-manager"}
	one := int32(1)
	_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MeshManager,
			Namespace: namespace,
			Labels:    m,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: metav1.SetAsLabelSelector(m),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      m,
					Annotations: m,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  "control-plane",
						Image: MeshControlPlaneImage,
						Env: []v1.EnvVar{{
							Name: "NAMESPACE",
							ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						}},
						// TODO: get image pull policy from config
						ImagePullPolicy: v1.PullAlways,
					}},
					ServiceAccountName: MeshManager,
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	// create service if not exists
	_, err = clientset.CoreV1().Services(namespace).Create(ctx, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MeshManager,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Protocol:   v1.ProtocolTCP,
				Port:       18000,
				TargetPort: intstr.FromInt(18000),
			}},
			Selector: m,
			Type:     v1.ServiceTypeClusterIP,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func patchOriginWorkloads(um *unstructured.Unstructured, podTemplate *v1.PodTemplateSpec, path string, client *clientgoutils.ClientGoUtils) error {
	jsonObj, err := um.MarshalJSON()
	if err != nil {
		return errors.WithStack(err)
	}

	pathItems := strings.Split(path, "/")
	pis := make([]string, 0)
	for _, item := range pathItems {
		if item != "" {
			pis = append(pis, item)
		}
	}

	pss, _ := json.Marshal(podTemplate)
	templatePath := strings.Join(pis, ".")
	jsonStr, err := sjson.SetRaw(string(jsonObj), templatePath, string(pss))
	if err != nil {
		return errors.WithStack(err)
	}

	infos, err := client.GetResourceInfoFromString(jsonStr, true)
	if err != nil {
		return err
	}

	if len(infos) != 1 {
		return errors.New(fmt.Sprintf("ResourceInfo' num is %d(not 1?)", len(infos)))
	}

	log.Infof("Creating %s(%v)", infos[0].Name, infos[0].Object.GetObjectKind().GroupVersionKind())
	return client.ApplyResourceInfo(infos[0], nil)
}
