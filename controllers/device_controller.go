/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	netboxv1 "github.com/networkop/declarative-netbox/api/v1"
	"github.com/networkop/declarative-netbox/netbox"
)

// DeviceReconciler reconciles a Device object
type DeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	netbox *netbox.NetboxServer
}

type DeviceReconcilerOptions struct {
	NetboxURL   string
	NetboxToken string
}

var retryInterval = time.Second * 5

//+kubebuilder:rbac:groups=netbox.networkop.co.uk,resources=devices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.networkop.co.uk,resources=devices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.networkop.co.uk,resources=devices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Device object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *DeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.V(1).Info("Reconcile", "req", req)

	var dev netboxv1.Device
	if err := r.Get(ctx, req.NamespacedName, &dev); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(&dev, netboxv1.DeviceFinalizer) {
		controllerutil.AddFinalizer(&dev, netboxv1.DeviceFinalizer)
		if err := r.Update(ctx, &dev); err != nil {
			log.Error(err, "unable to register finalizer")
			return ctrl.Result{}, err
		}
	}

	// handle deletion
	if !dev.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, dev)
	}

	// checking if the spec has changed
	if dev.Status.ObservedGeneration == dev.Generation {
		log.V(1).Info("Requed object after status update. Doing nothing")
		return ctrl.Result{}, nil
	}

	// handle create/update
	dev, result, err := r.reconcile(ctx, dev)

	// Generation only changes after updates to spec
	if dev.Status.ObservedGeneration != dev.Generation {
		dev.Status.ObservedGeneration = dev.Generation
		if err := r.Client.Status().Update(ctx, &dev); err != nil {
			log.Error(err, "unable to update Device status")
			return ctrl.Result{}, err
		}

	}

	log.V(1).Info("Reconciliation finished", "req", req)

	return result, err
}

func (r *DeviceReconciler) reconcile(ctx context.Context, dev netboxv1.Device) (netboxv1.Device, ctrl.Result, error) {
	log := logr.FromContext(ctx)
	log.V(1).Info("reconcile", "dev", dev)

	if err := r.netbox.Apply(ctx, &dev); err != nil {
		log.Error(err, "failed to r.netbox.Apply, retrying")
		return dev, ctrl.Result{RequeueAfter: retryInterval}, nil
	}

	return dev, ctrl.Result{}, nil
}

func (r *DeviceReconciler) reconcileDelete(ctx context.Context, dev netboxv1.Device) (ctrl.Result, error) {
	log := logr.FromContext(ctx)
	log.V(1).Info("reconcileDelete", "dev", dev)

	if err := r.netbox.Delete(ctx, &dev); err != nil {
		log.Error(err, "failed to r.netbox.Delete, retrying")
		return ctrl.Result{RequeueAfter: retryInterval}, err
	}

	// Remove finalizer to allow for the resource to be cleaned up
	controllerutil.RemoveFinalizer(&dev, netboxv1.DeviceFinalizer)
	if err := r.Update(ctx, &dev); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("Delete Reconciliation finished", "dev", dev)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceReconciler) SetupWithManager(mgr ctrl.Manager, opts DeviceReconcilerOptions) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.Device{}).
		Complete(r); err != nil {
		return err
	}

	r.netbox = netbox.NewNetboxServer(opts.NetboxURL, opts.NetboxToken)

	return nil
}
