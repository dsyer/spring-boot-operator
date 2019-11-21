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
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/dsyer/sample-controller/api/v1"
)

// MicroserviceReconciler reconciles a Microservice object
type MicroserviceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=spring.io,resources=springs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.io,resources=springs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

var (
	ownerKey = ".metadata.controller"
	apiGVStr = api.GroupVersion.String()
)

// Reconcile Business logic for controller
func (r *MicroserviceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("spring", req.NamespacedName)

	var spring api.Microservice
	if err := r.Get(ctx, req.NamespacedName, &spring); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Unable to fetch Spring")
		}
		return ctrl.Result{}, err
	}
	log.Info("Updating", "resource", spring)

	var services core.ServiceList
	var deployments apps.DeploymentList
	if err := r.List(ctx, &services, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name}); err != nil {
		log.Error(err, "Unable to list child Services")
		return ctrl.Result{}, err
	}
	if err := r.List(ctx, &deployments, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name}); err != nil {
		log.Error(err, "Unable to list child Deployments")
		return ctrl.Result{}, err
	}

	var service *core.Service
	var deployment *apps.Deployment

	log.Info("Found services", "services", len(services.Items))
	if len(services.Items) == 0 {
		var err error
		service, err = r.constructService(&spring)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "Unable to create Service for Spring", "service", service)
			return ctrl.Result{}, err
		}

		log.Info("Created Service for Spring", "service", service)
	} else {
		service = &services.Items[0]
	}
	if len(deployments.Items) == 0 {
		var err error
		deployment, err = r.constructDeployment(&spring)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Unable to create Deployment for Spring", "deployment", deployment)
			return ctrl.Result{}, err
		}

		log.Info("Created Deployments for Spring", "deployment", deployment)
	} else {
		deployment = &deployments.Items[0]
	}

	// TODO: What if the service or deployment doesn't match our spec? Recreate?
	spring.Status.ServiceName = service.GetName()
	spring.Status.Label = spring.Name
	spring.Status.Running = deployment.Status.AvailableReplicas > 0

	if err := r.Status().Update(ctx, &spring); err != nil {
		log.Error(err, "Unable to update Spring status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MicroserviceReconciler) constructService(spring *api.Microservice) (*core.Service, error) {
	service := &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"app": spring.Name},
			Name:      spring.Name,
			Namespace: spring.Namespace,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				core.ServicePort{
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Name:       "http",
				},
			},
			Selector: map[string]string{"app": spring.Name},
		},
	}
	if err := ctrl.SetControllerReference(spring, service, r.Scheme); err != nil {
		return nil, err
	}
	return service, nil
}

func (r *MicroserviceReconciler) constructDeployment(spring *api.Microservice) (*apps.Deployment, error) {
	deployment := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"app": spring.Name},
			Name:      spring.Name,
			Namespace: spring.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": spring.Name},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": spring.Name},
				},
			},
		},
	}
	spring.Spec.Pod.DeepCopyInto(&deployment.Spec.Template.Spec)
	container := findAppContainer(&deployment.Spec.Template.Spec)
	setUpAppContainer(container, *spring)
	r.Log.Info("Deploying", "deployment", deployment)
	if err := ctrl.SetControllerReference(spring, deployment, r.Scheme); err != nil {
		return nil, err
	}
	return deployment, nil
}

// Set up the app container, setting the image, adding probes etc.
func setUpAppContainer(container *core.Container, spring api.Microservice) {
	container.Name = "app"
	container.Image = spring.Spec.Image
	if spring.Spec.Actuators {
		if container.LivenessProbe == nil {
			container.LivenessProbe = &core.Probe{
				Handler: core.Handler{
					HTTPGet: &core.HTTPGetAction{
						Path: "/actuator/health",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       3,
			}
		}
		if container.ReadinessProbe == nil {
			container.ReadinessProbe = &core.Probe{
				Handler: core.Handler{
					HTTPGet: &core.HTTPGetAction{
						Path: "/actuator/info",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: 20,
				PeriodSeconds:       10,
			}
		}
	}

}

// Find the container that runs the app image
func findAppContainer(pod *core.PodSpec) *core.Container {
	var container *core.Container
	if len(pod.Containers) == 1 {
		container = &pod.Containers[0]
	}
	for _, candidate := range pod.Containers {
		if container.Name == "app" {
			container = &candidate
			break
		}
	}
	if container == nil {
		container = &core.Container{
			Name: "app",
		}
		pod.Containers = append(pod.Containers, *container)
		container = &pod.Containers[len(pod.Containers)-1]
	}
	return container
}

// SetupWithManager Utility method to set up manager
func (r *MicroserviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&apps.Deployment{}, ownerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		deployment := rawObj.(*apps.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		// ...make sure it's ours...
		if owner.APIVersion != apiGVStr || owner.Kind != "Microservice" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(&core.Service{}, ownerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		service := rawObj.(*core.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		// ...make sure it's ours...
		if owner.APIVersion != apiGVStr || owner.Kind != "Microservice" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Microservice{}).
		Owns(&core.Service{}).
		Owns(&apps.Deployment{}).
		Complete(r)
}
