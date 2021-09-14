/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package model

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Summary is a top-level container for holding NodeStats and PodStats.
type Summary struct {
	// Overall node stats.
	Node NodeStats `json:"node"`
	// Per-pod stats.
	Pods []PodStats `json:"pods"`
}

// NodeStats holds node-level unprocessed sample stats.
type NodeStats struct {
	// Reference to the measured Node.
	NodeName string `json:"nodeName"`
	// Stats pertaining to CPU resources.
	// +optional
	CPU CPUStats `json:"cpu,omitempty"`
	// Stats pertaining to memory (RAM) resources.
	// +optional
	Memory MemoryStats `json:"memory,omitempty"`
	// Stats pertaining to total usage of filesystem resources on the rootfs used by node k8s components.
	// NodeFs.Used is the total bytes used on the filesystem.
	// +optional
	Fs FsStats `json:"fs,omitempty"`
	// Stats about the underlying container runtime.
	// +optional
	Runtime RuntimeStats `json:"runtime,omitempty"`
}

// RuntimeStats are stats pertaining to the underlying container runtime.
type RuntimeStats struct {
	// Stats about the underlying filesystem where container images are stored.
	// This filesystem could be the same as the primary (root) filesystem.
	// Usage here refers to the total number of bytes occupied by images on the filesystem.
	// +optional
	ImageFs FsStats `json:"imageFs,omitempty"`
}

// PodStats holds pod-level unprocessed sample stats.
type PodStats struct {
	// Reference to the measured Pod.
	PodRef PodReference `json:"podRef"`
	// Stats pertaining to CPU resources consumed by pod cgroup (which includes all containers' resource usage and pod overhead).
	// +optional
	CPU *CPUStats `json:"cpu,omitempty"`
	// Stats pertaining to memory (RAM) resources consumed by pod cgroup (which includes all containers' resource usage and pod overhead).
	// +optional
	Memory *MemoryStats `json:"memory,omitempty"`
	// EphemeralStorage reports the total filesystem usage for the containers and emptyDir-backed volumes in the measured Pod.
	// +optional
	EphemeralStorage *FsStats `json:"ephemeral-storage,omitempty"`
}

// PodReference contains enough information to locate the referenced pod.
type PodReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// CPUStats contains data about CPU usage.
type CPUStats struct {
	// The time at which these stats were updated.
	Time metav1.Time `json:"time"`
	// Total CPU usage (sum of all cores) averaged over the sample window.
	// The "core" unit can be interpreted as CPU core-nanoseconds per second.
	// +optional
	UsageNanoCores uint64 `json:"usageNanoCores,omitempty"`
	// Cumulative CPU usage (sum of all cores) since object creation.
	// +optional
	UsageCoreNanoSeconds uint64 `json:"usageCoreNanoSeconds,omitempty"`
}

// MemoryStats contains data about memory usage.
type MemoryStats struct {
	// The time at which these stats were updated.
	Time metav1.Time `json:"time"`
	// Available memory for use.  This is defined as the memory limit - workingSetBytes.
	// If memory limit is undefined, the available bytes is omitted.
	// +optional
	AvailableBytes uint64 `json:"availableBytes,omitempty"`
	// Total memory in use. This includes all memory regardless of when it was accessed.
	// +optional
	UsageBytes uint64 `json:"usageBytes,omitempty"`
	// The amount of working set memory. This includes recently accessed memory,
	// dirty memory, and kernel memory. WorkingSetBytes is <= UsageBytes
	// +optional
	WorkingSetBytes uint64 `json:"workingSetBytes,omitempty"`
	// The amount of anonymous and swap cache memory (includes transparent
	// hugepages).
	// +optional
	RSSBytes uint64 `json:"rssBytes,omitempty"`
	// Cumulative number of minor page faults.
	// +optional
	PageFaults uint64 `json:"pageFaults,omitempty"`
	// Cumulative number of major page faults.
	// +optional
	MajorPageFaults uint64 `json:"majorPageFaults,omitempty"`
}

// FsStats contains data about filesystem usage.
type FsStats struct {
	// The time at which these stats were updated.
	Time metav1.Time `json:"time"`
	// AvailableBytes represents the storage space available (bytes) for the filesystem.
	// +optional
	AvailableBytes uint64 `json:"availableBytes,omitempty"`
	// CapacityBytes represents the total capacity (bytes) of the filesystems underlying storage.
	// +optional
	CapacityBytes uint64 `json:"capacityBytes,omitempty"`
	// UsedBytes represents the bytes used for a specific task on the filesystem.
	// This may differ from the total bytes used on the filesystem and may not equal CapacityBytes - AvailableBytes.
	// e.g. For ContainerStats.Rootfs this is the bytes used by the container rootfs on the filesystem.
	// +optional
	UsedBytes uint64 `json:"usedBytes,omitempty"`
	// InodesFree represents the free inodes in the filesystem.
	// +optional
	InodesFree uint64 `json:"inodesFree,omitempty"`
	// Inodes represents the total inodes in the filesystem.
	// +optional
	Inodes uint64 `json:"inodes,omitempty"`
	// InodesUsed represents the inodes used by the filesystem
	// This may not equal Inodes - InodesFree because this filesystem may share inodes with other "filesystems"
	// e.g. For ContainerStats.Rootfs, this is the inodes used only by that container, and does not count inodes used by other containers.
	InodesUsed uint64 `json:"inodesUsed,omitempty"`
}
