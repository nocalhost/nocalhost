/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

// GroupToTypeMap
// K: Workloads/Networks/Configurations
// V: deployments/statefulset
var GroupToTypeMap = []struct {
	K string
	V []string
}{
	{
		K: "Workloads",
		V: []string{
			"Deployment.v1.apps", // Kind.Version.Group
			"StatefulSet.v1.apps",
			"DaemonSet.v1.apps",
			"Job.v1.batch",
			"CronJob.v1beta1.batch",
			"CronJob.v1.batch",
			"Pod.v1.",
		},
	},
	{
		K: "Networks",
		V: []string{
			"Service.v1.",
			"Endpoints.v1.",
			"Ingress.v1.networking.k8s.io",
			"NetworkPolicy.v1.networking.k8s.io",
		},
	},
	{
		K: "Configurations",
		V: []string{
			"ConfigMap.v1.",
			"Secret.v1.",
			"HorizontalPodAutoscaler.v1.autoscaling",
			"ResourceQuota.v1.",
			"PodDisruptionBudget.v1.policy",
		},
	},
	{
		K: "Storages",
		V: []string{
			"PersistentVolume.v1.",
			"PersistentVolumeClaim.v1.",
			"StorageClass.v1.storage.k8s.io",
		},
	},
}
