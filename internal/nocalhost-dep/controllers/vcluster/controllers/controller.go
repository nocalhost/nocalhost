/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controllers

import (
	"context"
	"encoding/base64"
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
		}
	} else {
		if controllerutil.ContainsFinalizer(vc, helmv1alpha1.Finalizer) {
			lg.Info(fmt.Sprintf("deleting release: %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
			vc.Status.Phase = helmv1alpha1.Deleting
			if err := r.patchStatus(ctx, vc); err != nil {
				return ctrl.Result{}, err
			}
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

	state := ac.GetState(vc.GetReleaseName())
	switch state {
	case helper.ActionInstall:
		var opts []helper.InstallOption
		lg.Info(fmt.Sprintf("installing release: %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
		vc.Status.Phase = helmv1alpha1.Installing
		if err := r.patchStatus(ctx, vc); err != nil {
			return err
		}
		values, err := helper.ExtraValues(r.Config, vc)
		if err != nil {
			return err
		}
		_, err = ac.Install(vc.GetReleaseName(), vc.GetNamespace(), values, opts...)
		if err != nil {
			return err
		}
	case helper.ActionUpgrade:
		var opts []helper.UpgradeOption
		lg.Info(fmt.Sprintf("upgrading release: %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
		vc.Status.Phase = helmv1alpha1.Upgrading
		if err := r.patchStatus(ctx, vc); err != nil {
			return err
		}
		values, err := helper.ExtraValues(r.Config, vc)
		if err != nil {
			return err
		}
		_, err = ac.Upgrade(vc.GetReleaseName(), vc.GetNamespace(), values, opts...)
		if err != nil {
			return err
		}
	case helper.ActionError:
		lg.Error(errors.New(fmt.Sprintf("release %s/%s is in error state", vc.GetNamespace(), vc.GetReleaseName())), "")
		vc.Status.Phase = helmv1alpha1.Failed
		if err := r.patchStatus(ctx, vc); err != nil {
			return err
		}
		return errors.New(fmt.Sprintf("release %s/%s is in error state", vc.GetNamespace(), vc.GetReleaseName()))
	default:
		return errors.New("unexpected action state")
	}

	config, err := helper.NewAuthConfig(r.Config).Get(vc)
	if err != nil {
		return err
	}

	vc.Status.AuthConfig = base64.StdEncoding.EncodeToString([]byte(config))
	vc.Status.Phase = helmv1alpha1.Ready
	if err := r.patchStatus(ctx, vc); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) delete(ctx context.Context, vc *helmv1alpha1.VirtualCluster, ac helper.Actions) error {
	lg := log.FromContext(ctx)
	var opts []helper.UninstallOption
	_, err := ac.Uninstall(vc.GetReleaseName(), opts...)
	if errors.Cause(err) == driver.ErrReleaseNotFound {
		lg.Info(fmt.Sprintf("skip uninstall release: %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
		return nil
	}
	return err
}

func (r *Reconciler) patchStatus(ctx context.Context, vc *helmv1alpha1.VirtualCluster) error {
	lg := log.FromContext(ctx)
	key := client.ObjectKeyFromObject(vc)
	latest := &helmv1alpha1.VirtualCluster{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}
	lg.Info(fmt.Sprintf("update status for %s/%s", vc.GetNamespace(), vc.GetReleaseName()))
	return r.Client.Status().Patch(ctx, vc, client.MergeFrom(latest.DeepCopy()))
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.VirtualCluster{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Complete(r)
}
