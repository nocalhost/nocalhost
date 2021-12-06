/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Finalizer = "virtualcluster.helm.nocalhost.dev/finalizer"
)

// VirtualClusterSpec defines the desired state of VirtualCluster
type VirtualClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Helm HelmTemplate `json:"helm"`
}

type HelmTemplate struct {
	Chart ChartTemplate `json:"chart"`

	// +optional
	Values string `json:"values"`
}

type ChartTemplate struct {
	Name string `json:"name"`

	// +optional
	Version string `json:"version"`
	Repo    string `json:"repo"`
}

// VirtualClusterStatus defines the observed state of VirtualCluster
type VirtualClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VirtualCluster is the Schema for the virtualclusters API
type VirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterSpec   `json:"spec,omitempty"`
	Status VirtualClusterStatus `json:"status,omitempty"`
}

func (in *VirtualCluster) GetValues() string {
	return in.Spec.Helm.Values
}

func (in *VirtualCluster) GetReleaseName() string {
	return in.GetName()
}

func (in *VirtualCluster) GetChartName() string {
	return in.Spec.Helm.Chart.Name
}

func (in *VirtualCluster) GetChartVersion() string {
	return in.Spec.Helm.Chart.Version
}

func (in *VirtualCluster) GetChartRepo() string {
	return in.Spec.Helm.Chart.Repo
}

func (in *VirtualCluster) SetValues(values string) {
	in.Spec.Helm.Values = values
}

func (in *VirtualCluster) SetChartName(name string) {
	in.Spec.Helm.Chart.Name = name
}

func (in *VirtualCluster) SetChartVersion(version string) {
	in.Spec.Helm.Chart.Version = version
}

func (in *VirtualCluster) SetChartRepo(repo string) {
	in.Spec.Helm.Chart.Repo = repo
}

//+kubebuilder:object:root=true

// VirtualClusterList contains a list of VirtualCluster
type VirtualClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualCluster{}, &VirtualClusterList{})
}
