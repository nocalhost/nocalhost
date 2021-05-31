package model

import "nocalhost/internal/nhctl/appmeta"

type Namespace struct {
	Namespace   string             `json:"namespace" yaml:"namespace"`
	Application []*ApplicationInfo `json:"application" yaml:"application"`
}

type ApplicationInfo struct {
	Name string          `json:"name" yaml:"name"`
	Type appmeta.AppType `json:"type" yaml:"type"`
}
