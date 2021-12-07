/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

//type DuplicateDaemonSetController struct {
//	*Controller
//}
//
//func (d *DuplicateDaemonSetController) GetNocalhostDevContainerPod() (string, error) {
//	return d.GetDuplicateDevModePodName()
//}
//
//// ReplaceImage For DaemonSet, we don't replace the DaemonSet' image
//// but create a deployment with dev container instead
//func (d *DuplicateDaemonSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
//	return d.ReplaceDuplicateModeImage(ctx, ops)
//}
//
//func (d *DuplicateDaemonSetController) RollBack(reset bool) error {
//	return d.DuplicateModeRollBack()
//}
//
//// GetPodList
//// In DevMode, return pod list of generated Deployment.
//// Otherwise, return pod list of DaemonSet
//func (d *DuplicateDaemonSetController) GetPodList() ([]corev1.Pod, error) {
//	return d.GetDuplicateModePodList()
//}
