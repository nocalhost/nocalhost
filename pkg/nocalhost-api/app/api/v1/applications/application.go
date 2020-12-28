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

package applications

type CreateAppRequest struct {
	Context string `json:"context" binding:"required" example:"{\"application_url\":\"git@github.com:nocalhost/bookinfo.git\",\"application_name\":\"name\",\"source\":\"git/helm_repo\",\"install_type\":\"rawManifest/helm_chart\",\"resource_dir\":[\"manifest/templates\"],\"nocalhost_config\":\"base64encode(config_templates)\"}"`
	Status  *uint8 `json:"status" binding:"required"`
}

type UpdateApplicationInstallRequest struct {
	Status *uint64 `json:"status" binding:"required"`
}

// Application context struct
type ApplicationJsonContext struct {
	ApplicationName        string   `json:"application_name" validate:"required"`
	ApplicationURL         string   `json:"application_url" validate:"required"`
	ApplicationSource      string   `json:"source" validate:"required,oneof='git' 'helm_repo'"`
	ApplicationInstallType string   `json:"install_type" validate:"required,oneof='rawManifest' 'helm_chart'"`
	ApplicationSourceDir   []string `json:"resource_dir" validate:"required"`
}
