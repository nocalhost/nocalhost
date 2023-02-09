/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package handler

import (
	"encoding/json"
	_const "nocalhost/internal/nhctl/const"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type PodRouteConfig struct {
	LocalTunIP           string
	InboundPodTunIP      string
	TrafficManagerRealIP string
	Route                string
}

const VPN = "vpn"

func RemoveContainer(spec *v1.PodSpec) {
	for i := 0; i < len(spec.Containers); i++ {
		if spec.Containers[i].Name == VPN {
			spec.Containers = append(spec.Containers[:i], spec.Containers[i+1:]...)
		}
	}
}

func AddContainer(spec *v1.PodSpec, c *PodRouteConfig) {
	// remove vpn container if already exist
	for i := 0; i < len(spec.Containers); i++ {
		if spec.Containers[i].Name == VPN {
			spec.Containers = append(spec.Containers[:i], spec.Containers[i+1:]...)
		}
	}
	t := true
	zero := int64(0)
	spec.Containers = append(spec.Containers, v1.Container{
		Name:    VPN,
		Image:   _const.DefaultVPNImage,
		Command: []string{"/bin/sh", "-c"},
		Args: []string{
			"sysctl net.ipv4.ip_forward=1;" +
				"iptables -F;" +
				"iptables -P INPUT ACCEPT;" +
				"iptables -P FORWARD ACCEPT;" +
				"iptables -t nat -A PREROUTING ! -p icmp -j DNAT --to " + c.LocalTunIP + ";" +
				"iptables -t nat -A POSTROUTING ! -p icmp -j MASQUERADE;" +
				"sysctl -w net.ipv4.conf.all.route_localnet=1;" +
				"iptables -t nat -A OUTPUT -o lo ! -p icmp -j DNAT --to-destination " + c.LocalTunIP + ";" +
				"nhctl vpn serve -L 'tun://0.0.0.0:8421/" + c.TrafficManagerRealIP + ":8421?net=" + c.InboundPodTunIP + "&route=" + c.Route + "' --debug=true",
		},
		SecurityContext: &v1.SecurityContext{
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{
					"NET_ADMIN",
					//"SYS_MODULE",
				},
			},
			RunAsUser:  &zero,
			Privileged: &t,
		},
		Resources: v1.ResourceRequirements{
			Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU:    resource.MustParse("128m"),
				v1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU:    resource.MustParse("256m"),
				v1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		// TODO: get image pull policy from config
		ImagePullPolicy: v1.PullIfNotPresent,
	})
	if len(spec.PriorityClassName) == 0 {
		spec.PriorityClassName = "system-cluster-critical"
	}
}

func patch(spec v1.PodTemplateSpec, path []string) (removePatch []byte, restorePatch []byte) {
	type P struct {
		Op    string      `json:"op,omitempty"`
		Path  string      `json:"path,omitempty"`
		Value interface{} `json:"value,omitempty"`
	}
	var remove, restore []P
	for i := range spec.Spec.Containers {
		index := strconv.Itoa(i)
		readinessPath := strings.Join(append(path, "spec", "containers", index, "readinessProbe"), "/")
		livenessPath := strings.Join(append(path, "spec", "containers", index, "livenessProbe"), "/")
		startupPath := strings.Join(append(path, "spec", "containers", index, "startupProbe"), "/")
		remove = append(remove, P{
			Op:    "replace",
			Path:  readinessPath,
			Value: nil,
		}, P{
			Op:    "replace",
			Path:  livenessPath,
			Value: nil,
		}, P{
			Op:    "replace",
			Path:  startupPath,
			Value: nil,
		})
		restore = append(restore, P{
			Op:    "replace",
			Path:  readinessPath,
			Value: spec.Spec.Containers[i].ReadinessProbe,
		}, P{
			Op:    "replace",
			Path:  livenessPath,
			Value: spec.Spec.Containers[i].LivenessProbe,
		}, P{
			Op:    "replace",
			Path:  startupPath,
			Value: spec.Spec.Containers[i].StartupProbe,
		})
	}
	removePatch, _ = json.Marshal(remove)
	restorePatch, _ = json.Marshal(restore)
	return
}
