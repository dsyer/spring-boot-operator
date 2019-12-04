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
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/dsyer/spring-boot-operator/api/v1"
)

// MicroserviceReconciler reconciles a Microservice object
type MicroserviceReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=spring.io,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.io,resources=servicebindings/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.io,resources=microservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.io,resources=microservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
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

	var bindings api.ServiceBindingList
	if err := r.List(ctx, &bindings, client.InNamespace(corev1.NamespaceAll)); err != nil {
		log.Error(err, "Unable to list Bindings")
		// Not fatal
	}

	var micro api.Microservice
	if err := r.Get(ctx, req.NamespacedName, &micro); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Unable to fetch micro")
		}
		r.updateBindings(bindings, req)
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
	var bindingsToApply []api.ServiceBinding
	if len(micro.Spec.Bindings) > 0 {
		bindingsMap := map[string]api.ServiceBinding{}
		for _, binding := range bindings.Items {
			bindingsMap[fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)] = binding
			if binding.Namespace == micro.Namespace {
				bindingsMap[binding.Name] = binding
			}
		}
		bindingsToApply = findBindingsToApply(micro, bindingsMap)
		createdNewBindings := false
		for _, binding := range bindingsToApply {
			if _, ok := bindingsMap[fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)]; !ok {
				if err := r.Create(ctx, &binding); err != nil {
					log.Info("Unable to create default ServiceBinding", "binding", binding)
					msg := fmt.Sprintf("Unable to create service binding: %s", err)
					r.Recorder.Event(&micro, corev1.EventTypeWarning, "ErrResourceInvalid", msg)
					return ctrl.Result{}, err
				}
				createdNewBindings = true
				log.Info("Created ServiceBinding", "binding", binding)
				r.Recorder.Event(&micro, corev1.EventTypeNormal, "ResourceCreated", fmt.Sprintf("Created ServiceBinding: %s", binding.Name))
			}
		}
		if createdNewBindings {
			if err := r.List(ctx, &bindings, client.InNamespace(corev1.NamespaceAll)); err != nil {
				log.Error(err, "Unable to list Bindings")
				// Not fatal
			}
		}
	}

	r.copySecretsAndConfigMaps(bindingsToApply, req)

	var service *corev1.Service
	var deployment *apps.Deployment

	if len(deployments.Items) == 0 {
		var err error
		deployment, err = r.constructDeployment(bindingsToApply, &micro)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Unable to create Deployment for micro", "deployment", deployment)
			return ctrl.Result{}, err
		}

		log.Info("Created Deployments for micro", "deployment", deployment)
		r.Recorder.Event(&micro, corev1.EventTypeNormal, "DeploymentCreated", "Created Deployment")
	} else {
		// update if changed
		deployment = &deployments.Items[0]
		deployment = updateDeployment(deployment, bindingsToApply, &micro)
		if err := r.Update(ctx, deployment); err != nil {
			if apierrors.IsConflict(err) {
				log.Info("Unable to update Deployment: reason conflict. Will retry on next event.")
				err = nil
			} else {
				log.Error(err, "Unable to update Deployment for micro", "deployment", deployment)
				r.Recorder.Event(&micro, corev1.EventTypeWarning, "ErrInvalidResource", fmt.Sprintf("Could not create Deployment: %s", err))
			}
			return ctrl.Result{}, err
		}

		log.Info("Updated Deployments for micro", "deployment", deployment)
		r.Recorder.Event(&micro, corev1.EventTypeNormal, "DeploymentUpdated", "Updated Deployment")
	}

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
		r.Recorder.Event(&micro, corev1.EventTypeNormal, "ServiceCreated", "Created Service")
	} else {
		// update if changed
		service = &services.Items[0]
		service = updateService(service, &micro)
		if err := r.Update(ctx, service); err != nil {
			if apierrors.IsConflict(err) {
				log.Info("Unable to update Service: reason conflict. Will retry on next event.")
				err = nil
			} else {
				log.Error(err, "Unable to update Service for micro", "service", service)
			}
			return ctrl.Result{}, err
		}

		log.Info("Updated Service for micro", "service", service)
		r.Recorder.Event(&micro, corev1.EventTypeNormal, "ServiceUpdated", "Updated Service")
	}

	micro.Status.ServiceName = service.GetName()
	micro.Status.Label = micro.Name
	micro.Status.Running = deployment.Status.AvailableReplicas > 0

	if err := r.Status().Update(ctx, &micro); err != nil {
		if apierrors.IsConflict(err) {
			log.Info("Unable to update status: reason conflict. Will retry on next event.")
			err = nil
		} else {
			log.Error(err, "Unable to update micro status")
		}
		return ctrl.Result{}, err
	}
	if err := r.updateBindings(bindings, req); err != nil {
		if apierrors.IsConflict(err) {
			log.Info("Unable to update binding status: reason conflict. Will retry on next event.")
			err = nil
		} else {
			log.Error(err, "Unable to update binding status")
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MicroserviceReconciler) copySecretsAndConfigMaps(bindings []api.ServiceBinding, req ctrl.Request) error {
	if len(bindings) == 0 {
		return nil
	}
	ctx := context.Background()
	log := r.Log.WithValues("microservice", req.NamespacedName)
	var result error
	for _, binding := range bindings {
		if binding.Namespace == req.Namespace {
			// No need to copy config map in the same namespace
			continue
		}
		for _, volume := range binding.Spec.Template.Spec.Volumes {
			if volume.ConfigMap != nil {
				sourceName := volume.ConfigMap.Name
				var config corev1.ConfigMap
				if err := r.Get(ctx, client.ObjectKey{
					Namespace: req.Namespace,
					Name:      sourceName,
				}, &config); err != nil {
					if err := r.Get(ctx, client.ObjectKey{
						Namespace: binding.Namespace,
						Name:      sourceName,
					}, &config); err != nil {
						log.Info("Unable to obtain source ConfigMap", "namespace", binding.Namespace, "configmap", sourceName)
						continue
					}
					target := config.DeepCopy()
					target.ResourceVersion = ""
					target.Annotations["spring.io/servicebinding"] = fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)
					target.Namespace = req.Namespace
					if err := r.Create(ctx, target); err != nil {
						log.Error(err, "Unable to create ConfigMap")
						result = err
					}
				}
			}
			if volume.Secret != nil {
				sourceName := volume.Secret.SecretName
				var secret corev1.Secret
				if err := r.Get(ctx, client.ObjectKey{
					Namespace: req.Namespace,
					Name:      sourceName,
				}, &secret); err != nil {
					if err := r.Get(ctx, client.ObjectKey{
						Namespace: binding.Namespace,
						Name:      sourceName,
					}, &secret); err != nil {
						log.Info("Unable to obtain source Secret", "namespace", binding.Namespace, "secret", sourceName)
						continue
					}
					target := secret.DeepCopy()
					target.ResourceVersion = ""
					target.Annotations["spring.io/servicebinding"] = fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)
					target.Namespace = req.Namespace
					if err := r.Create(ctx, target); err != nil {
						log.Error(err, "Unable to create Secret")
						result = err
					}
				}
			}
		}
	}
	return result
}

func (r *MicroserviceReconciler) updateBindings(bindings api.ServiceBindingList, req ctrl.Request) error {
	if len(bindings.Items) == 0 {
		return nil
	}
	ctx := context.Background()
	log := r.Log.WithValues("microservice", req.NamespacedName)
	var micros api.MicroserviceList
	if err := r.List(ctx, &micros, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "Unable to list Microservices")
		return err
	}
	bounds := map[string][]string{}
	for _, micro := range micros.Items {
		for _, name := range micro.Spec.Bindings {
			key := name
			if !strings.Contains(name, "/") {
				key = fmt.Sprintf("default/%s", name)
			}
			bounds[key] = append(bounds[key], fmt.Sprintf("%s/%s", micro.Namespace, micro.Name))
		}
	}
	var result error
	names := []string{}
	for _, binding := range bindings.Items {
		bindingName := fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)
		if bound, ok := bounds[bindingName]; ok {
			binding.Status.Bound = bound
		} else {
			binding.Status.Bound = []string{}
		}
		names = append(names, bindingName)
		if err := r.Status().Update(ctx, &binding); err != nil {
			result = err
		}
	}
	log.Info("Updated binding statuses", "bindings", names)
	return result
}

func findBindingsToApply(micro api.Microservice, bindingsMap map[string]api.ServiceBinding) []api.ServiceBinding {
	var bindingsToApply = []api.ServiceBinding{}
	for _, name := range micro.Spec.Bindings {
		if binding, ok := bindingsMap[name]; ok {
			bindingsToApply = append(bindingsToApply, binding)
		} else {
			bindingsToApply = append(bindingsToApply, defaultBinding(name, micro))
		}
	}
	return bindingsToApply
}

func defaultBinding(name string, micro api.Microservice) api.ServiceBinding {
	initContainer := corev1.Container{
		Name: "env",
	}
	appContainer := corev1.Container{
		Name: "app",
	}
	binding := api.ServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: api.ServiceBindingSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
	}
	binding.Namespace = micro.Namespace
	binding.Spec.Template.Spec.Volumes = createVolumes(name, micro.Name)
	setUpInitContainer(&initContainer, name)
	location := "/etc/config/"
	addVolumeMount(&appContainer, location)
	addEnvVars(&binding.Spec, location)
	binding.Spec.Template.Spec.Containers = []corev1.Container{
		appContainer,
	}
	binding.Spec.Template.Spec.InitContainers = []corev1.Container{
		initContainer,
	}
	return binding
}

func addEnvVars(spec *api.ServiceBindingSpec, location string) {
	locations := []string{"classpath:/", "file://" + location}
	env := spec.Env
	env = setEnvVars(env, "CNB_BINDINGS", "/config/bindings")
	env = setEnvVarsMulti(env, "SPRING_CONFIG_LOCATION", locations)
	spec.Env = env
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

func updateService(service *corev1.Service, micro *api.Microservice) *corev1.Service {
	return service
}

func (r *MicroserviceReconciler) constructService(micro *api.Microservice) (*corev1.Service, error) {
	service := createService(micro)
	if err := ctrl.SetControllerReference(micro, service, r.Scheme); err != nil {
		return nil, err
	}
	return service, nil
}

func createDeployment(bindings []api.ServiceBinding, micro *api.Microservice) *apps.Deployment {
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
			Template: corev1.PodTemplateSpec{},
		},
	}
	deployment = updateDeployment(deployment, bindings, micro)
	return deployment
}

func updateDeployment(deployment *apps.Deployment, bindings []api.ServiceBinding, micro *api.Microservice) *apps.Deployment {
	defaults := findAppContainer(&micro.Spec.Template.Spec)
	if len(micro.Spec.Template.Spec.Containers) == 1 && defaults.Name == "" {
		// If there is only one container and it is anonymous, then it is the app container
		micro.Spec.Template.Spec.Containers[0].Name = "app"
		defaults.Name = "app"
	}
	for _, binding := range bindings {
		mergePodTemplates(binding.Spec.Template, &deployment.Spec.Template)
	}
	mergePodTemplates(micro.Spec.Template, &deployment.Spec.Template)
	container := findAppContainer(&deployment.Spec.Template.Spec)
	setUpAppContainer(container, *micro)
	// Reset all env vars so any deletions get picked up in the merge
	container.Env = defaults.Env
	mergeEnvVars(container, bindings)
	addProfiles(container, micro.Spec)
	if deployment.Spec.Template.ObjectMeta.Labels == nil {
		deployment.Spec.Template.ObjectMeta.Labels = map[string]string{}
	}
	deployment.Spec.Template.ObjectMeta.Labels["app"] = micro.Name
	return deployment
}

func mergeEnvVars(container *corev1.Container, bindings []api.ServiceBinding) {
	singles := map[string]string{}
	multis := map[string][]string{}
	for _, binding := range bindings {
		for _, value := range binding.Spec.Env {
			if len(value.Values) > 0 {
				multis[value.Name] = append(multis[value.Name], value.Values...)
			} else if value.Value != "" {
				singles[value.Name] = value.Value
			}
		}
	}
	for k, v := range multis {
		multis[k] = unique(v)
	}
	env := container.Env
	for _, value := range env {
		name := value.Name
		if singles[name] != "" {
			// Default to container spec if value is set there
			delete(singles, name)
			continue
		}
		// Append bindings onto CSV values supplied in container
		multis[name] = unique(append(strings.Split(value.Value, ","), multis[name]...))
	}
	for key, value := range singles {
		env = setEnvVar(env, key, value)
	}
	for key, value := range multis {
		env = setEnvVar(env, key, strings.Join(value, ","))
	}
	container.Env = env
}

func addProfiles(container *corev1.Container, spec api.MicroserviceSpec) {
	if len(spec.Profiles) > 0 {
		container.Env = setEnvVar(container.Env, "SPRING_PROFILES_ACTIVE", strings.Join(spec.Profiles, ","))
	}
}

func setEnvVar(values []corev1.EnvVar, name string, value string) []corev1.EnvVar {
	var env corev1.EnvVar
	var index int
	for index, env = range values {
		if env.Name == name {
			env.Value = value
			values[index] = env
			break
		}
	}
	if env.Name != name {
		env.Name = name
		env.Value = value
		values = append(values, env)
	}
	return values
}

func setEnvVars(values []api.EnvVar, name string, value string) []api.EnvVar {
	var env api.EnvVar
	for _, env = range values {
		if env.Name == name {
			env.Value = value
			break
		}
	}
	if env.Name != name {
		env.Name = name
		env.Value = value
		values = append(values, env)
	}
	return values
}

func setEnvVarsMulti(values []api.EnvVar, name string, value []string) []api.EnvVar {
	var env api.EnvVar
	for _, env = range values {
		if env.Name == name {
			env.Values = unique(append(env.Values, value...))
			break
		}
	}
	if env.Name != name {
		env.Name = name
		env.Values = value
		env.Value = ""
		values = append(values, env)
	}
	return values
}

func unique(values []string) []string {
	sifted := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if _, ok := sifted[value]; ok {
			continue
		}
		sifted[value] = true
		result = append(result, value)
	}
	return result
}

func addVolumeMount(container *corev1.Container, location string) {
	mounts := container.VolumeMounts
	locator := map[string]corev1.VolumeMount{}
	for _, volume := range mounts {
		locator[volume.Name] = volume
	}
	if _, ok := locator["config"]; !ok {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "config",
			MountPath: location,
		})
	}
	container.VolumeMounts = mounts
}

func createVolumes(binding string, config string) []corev1.Volume {
	volumes := []corev1.Volume{}
	name := fmt.Sprintf("%s-metadata", binding)
	volumes = append(volumes, corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		},
	})
	volumes = append(volumes, corev1.Volume{
		Name: fmt.Sprintf("%s-secret", binding),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-secret", binding),
			},
		},
	})
	volumes = append(volumes, corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	return volumes
}

// Create a Deployment for the microservice application
func (r *MicroserviceReconciler) constructDeployment(bindings []api.ServiceBinding, micro *api.Microservice) (*apps.Deployment, error) {
	deployment := createDeployment(bindings, micro)
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
						Path: "/actuator/info",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       10,
			}
		}
		if container.ReadinessProbe == nil {
			container.ReadinessProbe = &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/actuator/health",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: 10,
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
	} else {
		for _, candidate := range pod.Containers {
			if candidate.Name == "app" {
				container = &candidate
				break
			}
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

// Set up the init container, setting the image, adding volumes etc.
func setUpInitContainer(container *corev1.Container, binding string) {
	container.Name = "env"
	container.Image = "dsyer/spring-boot-bindings"
	container.Args = []string{
		"-f", "/etc/config/application.properties", "/config/bindings",
	}
	mounts := container.VolumeMounts
	locator := map[string]corev1.VolumeMount{}
	for _, volume := range mounts {
		locator[volume.Name] = volume
	}
	if _, ok := locator["config"]; !ok {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "config",
			MountPath: "/etc/config",
		})
	}
	if _, ok := locator[fmt.Sprintf("%s-metadata", binding)]; !ok {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      fmt.Sprintf("%s-metadata", binding),
			MountPath: fmt.Sprintf("/config/bindings/%s/metadata", binding),
		},
			corev1.VolumeMount{
				Name:      fmt.Sprintf("%s-secret", binding),
				MountPath: fmt.Sprintf("/config/bindings/%s/secret", binding),
			})
	}
	container.VolumeMounts = mounts
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
