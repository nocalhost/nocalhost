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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
	"nocalhost/internal/nocalhost-dep/controllers/vcluster/helper"
)

// Reconciler reconciles a VirtualCluster object
type Reconciler struct {
	client.Client
	Config *rest.Config
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=helm.nocalhost.dev,resources=virtualclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helm.nocalhost.dev,resources=virtualclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=helm.nocalhost.dev,resources=virtualclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)

	vc := &helmv1alpha1.VirtualCluster{}
	if err := r.Get(ctx, req.NamespacedName, vc); err != nil {
		lg.Error(err, "unable to fetch VirtualCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ac, err := helper.NewActions(vc, r.Config)
	if err != nil {
		return ctrl.Result{}, err
	}

	if vc.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(vc, helmv1alpha1.Finalizer) {
			controllerutil.AddFinalizer(vc, helmv1alpha1.Finalizer)
			if err := r.Update(ctx, vc); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(vc, helmv1alpha1.Finalizer) {
			lg.Info(fmt.Sprintf("deleting release: %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
			if err := r.delete(ctx, vc, ac); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(vc, helmv1alpha1.Finalizer)
			if err := r.Update(ctx, vc); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := r.reconcile(ctx, vc, ac); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, vc *helmv1alpha1.VirtualCluster, ac helper.Actions) error {
	lg := log.FromContext(ctx)

	chrt, err := ac.GetChart(vc.GetChartName())
	if err != nil {
		return err
	}

	state := ac.GetState(vc.GetReleaseName())
	switch state {
	case helper.ActionInstall:
		var opts []helper.InstallOption
		lg.Info(fmt.Sprintf("installing release: %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
		_, err := ac.Install(vc.GetReleaseName(), vc.GetNamespace(), chrt, opts...)
		if err != nil {
			return err
		}
	case helper.ActionUpgrade:
		var opts []helper.UpgradeOption
		lg.Info(fmt.Sprintf("upgrading release: %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
		_, err := ac.Upgrade(vc.GetReleaseName(), vc.GetNamespace(), chrt, opts...)
		if err != nil {
			return err
		}
	case helper.ActionError:
		lg.Error(errors.New(fmt.Sprintf("release %s/%s is in error state", vc.GetNamespace(), vc.GetReleaseName())), "")
		return errors.New(fmt.Sprintf("release %s/%s is in error state", vc.GetNamespace(), vc.GetReleaseName()))
	default:
		return errors.New("unexpected action state")
	}

	config, err := helper.NewAuthConfig(r.Config).Get(vc.GetReleaseName(), vc.GetNamespace())

	if err != nil {
		return err
	}
	fmt.Println(config)

	return nil
}

func (r *Reconciler) delete(ctx context.Context, vc *helmv1alpha1.VirtualCluster, ac helper.Actions) error {
	lg := log.FromContext(ctx)
	var opts []helper.UninstallOption
	_, err := ac.Uninstall(vc.GetReleaseName(), opts...)
	if errors.Cause(err) == driver.ErrReleaseNotFound {
		lg.Info(fmt.Sprintf("skip uninstall release: %s", vc.GetReleaseName()))
		return nil
	}
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.VirtualCluster{}).
		WithEventFilter(pred).
		Complete(r)
}
