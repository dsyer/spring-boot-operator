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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webappv1 "github.com/dsyer/sample-controller/api/v1"
)

// SpringReconciler reconciles a Spring object
type SpringReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=webapp.spring.io,resources=springs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.spring.io,resources=springs/status,verbs=get;update;patch

// Reconcile Business logic for controller
func (r *SpringReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("spring", req.NamespacedName)

	// your logic here
	// client.IgnoreNotFound()

	return ctrl.Result{}, nil
}

// SetupWithManager Utility method to set up manager
func (r *SpringReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Spring{}).
		Complete(r)
}
