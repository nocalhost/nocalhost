package item

import "nocalhost/internal/nhctl/profile"

type Result struct {
	Namespace   string `json:"namespace" yaml:"namespace"`
	Application []App  `json:"application" yaml:"application"`
}

type App struct {
	Name   string  `json:"name" yaml:"name"`
	Groups []Group `json:"group" yaml:"group"`
}

type Group struct {
	GroupName string     `json:"type" yaml:"type"`
	List      []Resource `json:"resource" yaml:"resource"`
}

type Resource struct {
	Name string `json:"name" yaml:"name"`
	List []Item `json:"list" yaml:"list"`
}

type Item struct {
	Metadata    interface{}           `json:"info,omitempty" yaml:"info"`
	Description *profile.SvcProfileV2 `json:"description,omitempty" yaml:"description"`
}
