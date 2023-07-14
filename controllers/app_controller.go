/*
Copyright 2023.

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
	"app-operator/controllers/utils"
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"app-operator/api/v1"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ingress.baiding.tech,resources=apps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.baiding.tech,resources=apps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ingress.baiding.tech,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the App object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	app := &v1.App{}

	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	newDeployment := utils.NewDeployment(app)
	if err := controllerutil.SetControllerReference(app, newDeployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	oldDeployment := &appsv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, oldDeployment); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, newDeployment); err != nil {
				logger.Error(err, "create deploy failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if err := r.Update(ctx, newDeployment); err != nil {
			return ctrl.Result{}, err
		}
	}

	newService := utils.NewService(app)
	if err := controllerutil.SetControllerReference(app, newService, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	oldService := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, oldService); err != nil {
		if errors.IsNotFound(err) && app.Spec.ServiceConfig.EnableService {
			if err := r.Create(ctx, newService); err != nil {
				logger.Error(err, "create Service failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if app.Spec.ServiceConfig.EnableService {
			clusterIP := oldService.Spec.ClusterIP
			oldService.Spec = newService.Spec
			oldService.Spec.ClusterIP = clusterIP
			if err := r.Update(ctx, oldService); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err := r.Delete(ctx, newService); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if !app.Spec.ServiceConfig.EnableService {
		return ctrl.Result{}, nil
	}
	newIngress := utils.NewIngress(app)
	if err := controllerutil.SetControllerReference(app, newIngress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	oldIngress := &netv1.Ingress{}
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, oldIngress); err != nil {
		if errors.IsNotFound(err) && app.Spec.IngressConfig.EnableIngress {
			if err := r.Create(ctx, newIngress); err != nil {
				logger.Error(err, "create Ingress failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if app.Spec.IngressConfig.EnableIngress {
			if err := r.Update(ctx, newIngress); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err := r.Delete(ctx, newIngress); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.App{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}
