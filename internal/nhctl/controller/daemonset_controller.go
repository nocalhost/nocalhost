/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

//type DaemonSetController struct {
//	*Controller
//}
//
//func (d *DaemonSetController) GetNocalhostDevContainerPod() (string, error) {
//	return d.GetDevModePodName()
//}
//
//// ReplaceImage For DaemonSet, we don't replace the DaemonSet' image
//// but create a deployment with dev container instead
//func (d *DaemonSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
//	return d.PatchDevModeManifest(ctx, ops)
//}
//
//func (d *DaemonSetController) RollBack(reset bool) error {
//	return d.RollbackFromAnnotation()
//}
//
//// GetPodList
//// In DevMode, return pod list of generated Deployment.
//// Otherwise, return pod list of DaemonSet
//func (d *DaemonSetController) GetPodList() ([]corev1.Pod, error) {
//	return d.Controller.GetPodList()
//}
