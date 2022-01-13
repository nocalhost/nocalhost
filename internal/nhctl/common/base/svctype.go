/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package base

type SvcType string

const (
	Deployment  SvcType = "deployment"
	StatefulSet SvcType = "statefulset"
	DaemonSet   SvcType = "daemonset"
	Job         SvcType = "job"
	CronJob     SvcType = "cronjob"
	Pod         SvcType = "pod"
	DEPLOYMENT  SvcType = "D"
)

// Alias For compatibility with meta
func (s SvcType) Alias() SvcType {
	if s == Deployment {
		return DEPLOYMENT
	}
	return s
}

func (s SvcType) Origin() SvcType {
	if s == DEPLOYMENT {
		return Deployment
	}
	return s
}

func (s SvcType) String() string {
	return string(s)
}
