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
	"fmt"
	"strings"

	"github.com/vmware-labs/reconciler-runtime/reconcilers"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	api "github.com/dsyer/spring-boot-operator/api/v1"
)

// +kubebuilder:rbac:groups=spring.io,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.io,resources=servicebindings/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.io,resources=microservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.io,resources=microservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

var (
	ownerKey = ".metadata.controller"
	apiGVStr = api.GroupVersion.String()
)

// MicroserviceReconciler reconciles a Microservice object
func MicroserviceReconciler(c reconcilers.Config) *reconcilers.ParentReconciler {
	c.Log = c.Log.WithName("Microservice")

	return &reconcilers.ParentReconciler{
		Type: &api.Microservice{},
		SubReconcilers: []reconcilers.SubReconciler{
			DeploymentReconciler(c),
			ServiceReconciler(c),
		},

		Config: c,
	}
}

// DeploymentReconciler creates a new Deployment if needed
func DeploymentReconciler(c reconcilers.Config) reconcilers.SubReconciler {
	c.Log = c.Log.WithName("Deployment")

	return &reconcilers.ChildReconciler{
		Config:        c,
		ParentType:    &api.Microservice{},
		ChildType:     &apps.Deployment{},
		ChildListType: &apps.DeploymentList{},

		DesiredChild: func(micro *api.Microservice) (*apps.Deployment, error) {
			return createDeployment([]api.ServiceBinding{}, micro), nil
		},

		ReflectChildStatusOnParent: func(micro *api.Microservice, child *apps.Deployment, err error) {
			if err != nil {
				return
			}
			if child == nil {
				micro.Status.Running = false
			} else {
				micro.Status.Running = child.Status.AvailableReplicas > 0
			}
		},

		MergeBeforeUpdate: func(current, desired *apps.Deployment) {
			current.Labels = desired.Labels
			current.Spec = desired.Spec
		},

		SemanticEquals: func(a1, a2 *apps.Deployment) bool {
			return equality.Semantic.DeepEqual(a1.Spec, a2.Spec) &&
				equality.Semantic.DeepEqual(a1.Labels, a2.Labels)
		},

		IndexField: ".metadata.deploymentController",

		Sanitize: func(child *apps.Deployment) interface{} {
			return child.Spec
		},
	}
}

// ServiceReconciler creates a new Service if needed
func ServiceReconciler(c reconcilers.Config) reconcilers.SubReconciler {
	c.Log = c.Log.WithName("Service")

	return &reconcilers.ChildReconciler{
		Config:        c,
		ParentType:    &api.Microservice{},
		ChildType:     &corev1.Service{},
		ChildListType: &corev1.ServiceList{},

		DesiredChild: func(micro *api.Microservice) (*corev1.Service, error) {
			return createService(micro), nil
		},

		ReflectChildStatusOnParent: func(micro *api.Microservice, child *corev1.Service, err error) {
			if err != nil {
				return
			}
			if child != nil {
				micro.Status.ServiceName = child.ObjectMeta.Name
			}
		},

		HarmonizeImmutableFields: func(current, desired *corev1.Service) {
			desired.Spec.ClusterIP = current.Spec.ClusterIP
		},

		MergeBeforeUpdate: func(current, desired *corev1.Service) {
			current.Labels = desired.Labels
			current.Spec = desired.Spec
		},

		SemanticEquals: func(a1, a2 *corev1.Service) bool {
			return equality.Semantic.DeepEqual(a1.Spec, a2.Spec) &&
				equality.Semantic.DeepEqual(a1.Labels, a2.Labels)
		},

		IndexField: ".metadata.serviceController",

		Sanitize: func(child *corev1.Service) interface{} {
			return child.Spec
		},
	}
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
				{
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
	deployment.Spec.Template = *updatePodTemplate(&deployment.Spec.Template, bindings, micro)
	return deployment
}

func updatePodTemplate(template *corev1.PodTemplateSpec, bindings []api.ServiceBinding, micro *api.Microservice) *corev1.PodTemplateSpec {
	defaults := findAppContainer(&micro.Spec.Template.Spec)
	if len(micro.Spec.Template.Spec.Containers) == 1 && defaults.Name == "" {
		// If there is only one container and it is anonymous, then it is the app container
		micro.Spec.Template.Spec.Containers[0].Name = "app"
		defaults.Name = "app"
	}
	for _, binding := range bindings {
		mergeResources(binding.Spec.Template, template)
	}
	mergeResources(micro.Spec.Template, template)
	container := findAppContainer(&template.Spec)
	setUpAppContainer(container, *micro)
	// Reset all env vars so any deletions get picked up in the merge
	container.Env = defaults.Env
	mergeEnvVars(container, bindings)
	addProfiles(container, micro.Spec)
	if template.ObjectMeta.Labels == nil {
		template.ObjectMeta.Labels = map[string]string{}
	}
	template.ObjectMeta.Labels["app"] = micro.Name
	return template
}

// Set up the app container, setting the image, adding args etc.
func setUpAppContainer(container *corev1.Container, micro api.Microservice) {
	container.Name = "app"
	container.Image = micro.Spec.Image
	if len(micro.Spec.Args) > 0 {
		container.Args = micro.Spec.Args
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

func defaultBinding(name string, micro api.Microservice) api.ServiceBinding {
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
	if name == "actuators" {
		appContainer.LivenessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/actuator/info",
					Port: intstr.FromInt(8080),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       11,
		}
		appContainer.ReadinessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/actuator/health",
					Port: intstr.FromInt(8080),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       13,
		}
		binding.Spec.Template.Spec.Containers = []corev1.Container{
			appContainer,
		}
		return binding
	}
	binding.Spec.Template.Spec.Volumes = createVolumes(name, micro.Name)
	location := "/etc/config/"
	addVolumeMount(&appContainer, location)
	addEnvVars(&binding.Spec, location)
	binding.Spec.Template.Spec.Containers = []corev1.Container{
		appContainer,
	}
	initContainer := corev1.Container{
		Name: "env",
	}
	setUpInitContainer(&initContainer, name)
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
