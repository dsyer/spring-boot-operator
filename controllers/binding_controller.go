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
	"strings"

	"github.com/vmware-labs/reconciler-runtime/reconcilers"
	"github.com/vmware-labs/reconciler-runtime/tracker"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/source"

	api "github.com/dsyer/spring-boot-operator/api/v1"
)

// ServiceBindingReconciler reconciles a ServiceBinding object
func ServiceBindingReconciler(c reconcilers.Config) *reconcilers.ParentReconciler {
	c.Log = c.Log.WithName("ServiceBinding")

	return &reconcilers.ParentReconciler{
		Type: &api.ServiceBinding{},
		SubReconcilers: []reconcilers.SubReconciler{
			BindingDeploymentReconciler(c),
		},

		Config: c,
	}
}

// BindingDeploymentReconciler updates Microservices when bindings change
func BindingDeploymentReconciler(c reconcilers.Config) reconcilers.SubReconciler {
	c.Log = c.Log.WithName("ServiceBinding")

	return &reconcilers.SyncReconciler{

		Sync: func(ctx context.Context, binding api.ServiceBinding) error {
			for _, name := range binding.Status.Bound {
				var micro api.Microservice
				namespace := binding.Namespace
				if strings.Contains(name, "/") {
					namespaced := strings.Split(name, "/")
					name = namespaced[1]
					namespace = namespaced[0]
				}
				key := types.NamespacedName{
					Namespace: namespace,
					Name:      name,
				}
				c.Tracker.Track(
					tracker.NewKey(v1.SchemeGroupVersion.WithKind("Microservice"), key),
					types.NamespacedName{Namespace: namespace, Name: name},
				)
				if err := c.Get(ctx, key, &micro); err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}
					return err
				}
				if len(micro.Spec.Template.Spec.Containers) == 0 {
					c.Log.Info("Empty containers.")
					micro.Spec.Template.Spec.Containers = []v1.Container{}
				}
				annos := micro.ObjectMeta.GetAnnotations()
				// Add an annotation to jog the API server to update the deployment if necessary
				if annos["spring.io/active"] == "red" {
					annos["spring.io/active"] = "black"
				} else {
					annos["spring.io/active"] = "red"
				}
				if err := c.Update(ctx, &micro); err != nil {
					if apierrors.IsConflict(err) {
						c.Log.Info("Unable to update Microservice: reason conflict. Will retry on next event.")
						err = nil
					} else {
						c.Log.Error(err, "Unable to update Microservice for binding", "micro", micro)
					}
					return err
				}
			}

			return nil
		},

		Config: c,

		Setup: func(mgr reconcilers.Manager, bldr *reconcilers.Builder) error {
			bldr.Watches(&source.Kind{Type: &api.Microservice{}}, reconcilers.EnqueueTracked(&api.Microservice{}, c.Tracker, c.Scheme))
			return nil
		},
	}
}
