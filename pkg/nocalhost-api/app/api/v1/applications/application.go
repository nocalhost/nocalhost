/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package applications

type CreateAppRequest struct {
	Context string `json:"context" binding:"required" example:"{\"application_url\":\"git@github.com:nocalhost/bookinfo.git\",\"application_name\":\"name\",\"source\":\"git/helm_repo\",\"install_type\":\"rawManifest/helm_chart\",\"resource_dir\":[\"manifest/templates\"],\"nocalhost_config_raw\":\"base64encode(config_templates)\",\"nocalhost_config_path\":\"./nocalhost/config.yaml\"}"`
	Status  *uint8 `json:"status" binding:"required"`
	Public  *uint8 `json:"public"`
}

type AppPublicSwitchRequest struct {
	Public *uint8 `json:"public" binding:"required"`
}

type UpdateApplicationInstallRequest struct {
	Status *uint64 `json:"status" binding:"required"`
}

const (
	HelmGit  = "helmGit"
	HelmRepo = "helmRepo"
	// same as ManifestGit
	Manifest       = "rawManifest"
	ManifestGit    = "rawManifestGit"
	ManifestLocal  = "rawManifestLocal"
	HelmLocal      = "helmLocal"
	KustomizeGit   = "kustomizeGit"
	KustomizeLocal = "kustomizeLocal"

	// source
	SourceGit      = "git"
	SourceHelmRepo = "helm_repo"
	SourceLocal    = "local"

	// install type
	ITRawManifest      = "rawManifest"
	ITHelmChart        = "helm_chart"
	ITRawManifestLocal = "rawManifestLocal"
	ITHelmLocal        = "helmLocal"
	ITKustomize        = "kustomize"
	ITKustomizeLocal   = "kustomizeLocal"
)

// Application context struct
type ApplicationJsonContext struct {
	ApplicationName        string   `json:"application_name" validate:"required"`
	ApplicationURL         string   `json:"application_url" validate:"required"`
	ApplicationSource      string   `json:"source" validate:"required,oneof='git' 'helm_repo' 'local'"`
	ApplicationInstallType string   `json:"install_type" validate:"required,oneof='rawManifest' 'helm_chart' 'rawManifestLocal' 'helmLocal' 'kustomize' 'kustomizeLocal'"`
	ApplicationSourceDir   []string `json:"resource_dir" validate:"required"`
	NocalhostRawConfig     string   `json:"nocalhost_config_raw"`
	NocalhostConfigPath    string   `json:"nocalhost_config_path"`
}

type ApplicationListQuery struct {
	UserId *uint64 `form:"user_id"`
}
