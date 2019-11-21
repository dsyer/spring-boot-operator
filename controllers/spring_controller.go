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
	corev1 "k8s.io/api/core/v1"
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

// +kubebuilder:rbac:groups=micro.io,resources=microservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=micro.io,resources=microservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

var (
	ownerKey = ".metadata.controller"
	apiGVStr = api.GroupVersion.String()
)

// Reconcile Business logic for controller
func (r *MicroserviceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("microservice", req.NamespacedName)

	var micro api.Microservice
	if err := r.Get(ctx, req.NamespacedName, &micro); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Unable to fetch micro")
		}
		return ctrl.Result{}, err
	}
	log.Info("Updating", "resource", micro)

	var services corev1.ServiceList
	var deployments apps.DeploymentList
	if err := r.List(ctx, &services, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name}); err != nil {
		log.Error(err, "Unable to list child Services")
		return ctrl.Result{}, err
	}
	if err := r.List(ctx, &deployments, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name}); err != nil {
		log.Error(err, "Unable to list child Deployments")
		return ctrl.Result{}, err
	}

	var service *corev1.Service
	var deployment *apps.Deployment

	log.Info("Found services", "services", len(services.Items))
	if len(services.Items) == 0 {
		var err error
		service, err = r.constructService(&micro)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "Unable to create Service for micro", "service", service)
			return ctrl.Result{}, err
		}

		log.Info("Created Service for micro", "service", service)
	} else {
		// TODO: update if changed
		service = &services.Items[0]
	}
	if len(deployments.Items) == 0 {
		var err error
		deployment, err = r.constructDeployment(&micro)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Unable to create Deployment for micro", "deployment", deployment)
			return ctrl.Result{}, err
		}

		log.Info("Created Deployments for micro", "deployment", deployment)
	} else {
		// TODO: update if changed
		deployment = &deployments.Items[0]
	}

	micro.Status.ServiceName = service.GetName()
	micro.Status.Label = micro.Name
	micro.Status.Running = deployment.Status.AvailableReplicas > 0

	if err := r.Status().Update(ctx, &micro); err != nil {
		log.Error(err, "Unable to update micro status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func createService(micro *api.Microservice) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"app": micro.Name},
			Name:      micro.Name,
			Namespace: micro.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Name:       "http",
				},
			},
			Selector: map[string]string{"app": micro.Name},
		},
	}
	return service
}

func (r *MicroserviceReconciler) constructService(micro *api.Microservice) (*corev1.Service, error) {
	service := createService(micro)
	if err := ctrl.SetControllerReference(micro, service, r.Scheme); err != nil {
		return nil, err
	}
	return service, nil
}

func createDeployment(micro *api.Microservice) *apps.Deployment {
	deployment := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"app": micro.Name},
			Name:      micro.Name,
			Namespace: micro.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": micro.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": micro.Name},
				},
			},
		},
	}
	micro.Spec.Pod.DeepCopyInto(&deployment.Spec.Template.Spec)
	container := findAppContainer(&deployment.Spec.Template.Spec)
	setUpAppContainer(container, *micro)
	return deployment
}

// Create a Deployment for the microservice application
func (r *MicroserviceReconciler) constructDeployment(micro *api.Microservice) (*apps.Deployment, error) {
	deployment := createDeployment(micro)
	r.Log.Info("Deploying", "deployment", deployment)
	if err := ctrl.SetControllerReference(micro, deployment, r.Scheme); err != nil {
		return nil, err
	}
	return deployment, nil
}

// Set up the app container, setting the image, adding probes etc.
func setUpAppContainer(container *corev1.Container, micro api.Microservice) {
	container.Name = "app"
	container.Image = micro.Spec.Image
	if micro.Spec.Actuators {
		if container.LivenessProbe == nil {
			container.LivenessProbe = &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/actuator/health",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       3,
			}
		}
		if container.ReadinessProbe == nil {
			container.ReadinessProbe = &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
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
func findAppContainer(pod *corev1.PodSpec) *corev1.Container {
	var container *corev1.Container
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
		container = &corev1.Container{
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
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Service{}, ownerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		service := rawObj.(*corev1.Service)
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
		Owns(&corev1.Service{}).
		Owns(&apps.Deployment{}).
		Complete(r)
}
