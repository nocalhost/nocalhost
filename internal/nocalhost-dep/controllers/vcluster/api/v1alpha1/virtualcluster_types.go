/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Finalizer      = "virtualcluster.helm.nocalhost.dev/finalizer"
	DefaultVersion = "0.4.5"
)

// VirtualClusterSpec defines the desired state of VirtualCluster
type VirtualClusterSpec struct {
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
	Version string `json:"version,omitempty"`
	Repo    string `json:"repo"`
}

// VirtualClusterStatus defines the observed state of VirtualCluster
type VirtualClusterStatus struct {
	// +optional
	Phase VirtualClusterPhase `json:"phase,omitempty"`
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`
	// +optional
	AuthConfig string `json:"authConfig,omitempty"`
	// +optional
	Manifest string `json:"manifest,omitempty"`
}

type VirtualClusterPhase string

const (
	Installing VirtualClusterPhase = "Installing"
	Upgrading  VirtualClusterPhase = "Upgrading"
	Ready      VirtualClusterPhase = "Ready"
	Failed     VirtualClusterPhase = "Failed"
	Deleting   VirtualClusterPhase = "Deleting"
	Unknown    VirtualClusterPhase = "Unknown"
)

type Condition struct {
	// +optional
	Type ConditionType `json:"type,omitempty"`
	// +optional
	Status corev1.ConditionStatus `json:"status,omitempty"`
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

type ConditionType string

type Conditions []Condition

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VirtualCluster is the Schema for the virtualclusters API
type VirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualClusterSpec `json:"spec,omitempty"`
	// +optional
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
