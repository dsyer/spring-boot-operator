/*

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

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/dsyer/spring-boot-operator/api/v1"
)

// ServiceBindingReconciler reconciles a ServiceBinding object
type ServiceBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile Business logic for controller
func (r *ServiceBindingReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("servicebinding", req.NamespacedName)

	var binding api.ServiceBinding
	if err := r.Get(ctx, req.NamespacedName, &binding); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Unable to fetch service binding")
		}
		return ctrl.Result{}, err
	}
	log.Info("Updating", "resource", binding.Name)

	for _, name := range binding.Status.Bound {
		log.Info("Updating", "micro", name)
		var micro api.Microservice
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: req.Namespace,
			Name:      name,
		}, &micro); err != nil {
			err = client.IgnoreNotFound(err)
			if err != nil {
				log.Error(err, "Unable to fetch micro service")
			}
			return ctrl.Result{}, err
		}
		if len(micro.Spec.Template.Spec.Containers) == 0 {
			log.Info("Empty containers.")
			micro.Spec.Template.Spec.Containers = []v1.Container{}
		}
		annos := micro.ObjectMeta.GetAnnotations()
		// Add an annotation to jog the API server to update the deployment if necessary
		if annos["spring.io/active"] == "red" {
			annos["spring.io/active"] = "black"
		} else {
			annos["spring.io/active"] = "red"
		}
		if err := r.Update(ctx, &micro); err != nil {
			if apierrors.IsConflict(err) {
				log.Info("Unable to update Microservice: reason conflict. Will retry on next event.")
				err = nil
			} else {
				log.Error(err, "Unable to update Microservice for binding", "micro", micro)
			}
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager Utility method to set up manager
func (r *ServiceBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.ServiceBinding{}).
		Complete(r)
}
