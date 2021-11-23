/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controllers

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"nocalhost/internal/nocalhost-dep/controllers/vcluster/helm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
)

// VirtualClusterReconciler reconciles a VirtualCluster object
type VirtualClusterReconciler struct {
	client.Client
	Config *rest.Config
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=helm.nocalhost.dev,resources=virtualclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helm.nocalhost.dev,resources=virtualclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=helm.nocalhost.dev,resources=virtualclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VirtualClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	vc := &helmv1alpha1.VirtualCluster{}
	if err := r.Get(ctx, req.NamespacedName, vc); err != nil {
		log.Error(err, "unable to fetch VirtualCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	hc, err := helm.NewClient(r.Config, vc.GetNamespace())
	if err != nil {
		return ctrl.Result{}, err
	}

	if vc.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(vc, helmv1alpha1.Finalizer) {
			controllerutil.AddFinalizer(vc, helmv1alpha1.Finalizer)
			if err := r.Update(ctx, vc); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(vc, helmv1alpha1.Finalizer) {
			if err := hc.UnInstall(vc); err != nil {
				if errors.Cause(err) != driver.ErrReleaseNotFound {
					return ctrl.Result{}, err
				}
				log.Error(errors.Cause(err), "skip uninstall")
			}

			controllerutil.RemoveFinalizer(vc, helmv1alpha1.Finalizer)
			if err := r.Update(ctx, vc); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if err := hc.SetValues(vc); err != nil {
		return ctrl.Result{}, err
	}

	chrt, err := hc.LoadChart(vc)
	if err != nil {
		return ctrl.Result{}, err
	}

	if hc.IsInstalled(vc) {
		log.Info(fmt.Sprintf("upgrade release: %s", vc.GetReleaseName()))
		_, err := hc.Upgrade(vc, chrt)
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("install release: %s", vc.GetReleaseName()))
	_, err = hc.Install(vc, chrt)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.VirtualCluster{}).
		Complete(r)
}
