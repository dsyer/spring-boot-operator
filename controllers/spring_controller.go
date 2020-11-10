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

	"github.com/vmware-labs/reconciler-runtime/reconcilers"
	"github.com/vmware-labs/reconciler-runtime/tracker"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

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
			DeploymentBindingReconciler(c),
			DeploymentReconciler(c),
			ServiceReconciler(c),
		},

		Config: c,
	}
}

// DeploymentBindingReconciler creates a new Deployment if needed
func DeploymentBindingReconciler(c reconcilers.Config) reconcilers.SubReconciler {
	c.Log = c.Log.WithName("DeploymentBinding")
	r := ContextReconciler{}
	config := ConfigMapReconciler(c, &r)
	secret := SecretReconciler(c, &r)
	return &reconcilers.SyncReconciler{

		Sync: func(ctx context.Context, micro *api.Microservice) error {
			bindingsToApply := findBindings(c, micro)
			for _, binding := range bindingsToApply {
				r.resetConfig()
				if binding.Namespace == micro.Namespace {
					continue
				}
				for _, volume := range binding.Spec.Template.Spec.Volumes {
					if volume.ConfigMap != nil {
						sourceName := volume.ConfigMap.Name
						var configMap corev1.ConfigMap
						if err := c.Get(ctx, client.ObjectKey{
							Namespace: micro.Namespace,
							Name:      sourceName,
						}, &configMap); err != nil {
							if err := c.Get(ctx, client.ObjectKey{
								Namespace: binding.Namespace,
								Name:      sourceName,
							}, &configMap); err != nil {
								c.Log.Info("Unable to obtain source ConfigMap", "namespace", binding.Namespace, "configmap", sourceName)
								continue
							}
							target := configMap.DeepCopy()
							target.ResourceVersion = ""
							target.Annotations["spring.io/servicebinding"] = fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)
							target.Namespace = micro.Namespace
							target.Name = fmt.Sprintf("%s-%s", micro.Name, volume.ConfigMap.Name)
							r.config = *target
							config.Reconcile(ctx, micro)
						}
					}
					if volume.Secret != nil {
						sourceName := volume.Secret.SecretName
						var source corev1.Secret
						if err := c.Get(ctx, client.ObjectKey{
							Namespace: micro.Namespace,
							Name:      sourceName,
						}, &source); err != nil {
							if err := c.Get(ctx, client.ObjectKey{
								Namespace: binding.Namespace,
								Name:      sourceName,
							}, &source); err != nil {
								c.Log.Info("Unable to obtain source Secret", "namespace", binding.Namespace, "secret", sourceName)
								continue
							}
							target := source.DeepCopy()
							target.ResourceVersion = ""
							target.Annotations["spring.io/servicebinding"] = fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)
							target.Namespace = micro.Namespace
							target.Name = fmt.Sprintf("%s-%s", micro.Name, volume.Secret.SecretName)
							r.secret = *target
							secret.Reconcile(ctx, micro)
						}
					}
				}
			}
			return nil
		},

		Config: c,

		Setup: func(mgr reconcilers.Manager, bldr *reconcilers.Builder) error {
			config.SetupWithManager(mgr, bldr)
			secret.SetupWithManager(mgr, bldr)
			return nil
		},
	}
}

// ContextReconciler like a ChildReconciler but with context
type ContextReconciler struct {
	config corev1.ConfigMap
	secret corev1.Secret
}

func (c *ContextReconciler) resetConfig() {
	c.secret = corev1.Secret{}
	c.config = corev1.ConfigMap{}
}

// ConfigMapReconciler creates a new ConfigMap if needed
func ConfigMapReconciler(c reconcilers.Config, r *ContextReconciler) reconcilers.SubReconciler {
	c.Log = c.Log.WithName("ConfigMap")
	return &reconcilers.ChildReconciler{
		Config:        c,
		ParentType:    &api.Microservice{},
		ChildType:     &corev1.ConfigMap{},
		ChildListType: &corev1.ConfigMapList{},

		DesiredChild: func(micro *api.Microservice) (*corev1.ConfigMap, error) {
			return &r.config, nil
		},

		ReflectChildStatusOnParent: func(micro *api.Microservice, child *corev1.ConfigMap, err error) {
			return
		},

		MergeBeforeUpdate: func(current, desired *corev1.ConfigMap) {
			current.Labels = desired.Labels
		},

		SemanticEquals: func(a1, a2 *corev1.ConfigMap) bool {
			return equality.Semantic.DeepEqual(a1.Labels, a2.Labels) && equality.Semantic.DeepEqual(a1.Data, a2.Data)
		},

		IndexField: ".metadata.serviceBindingConfigMap",

		Sanitize: func(child *corev1.ConfigMap) interface{} {
			return child.Data
		},
	}
}

// SecretReconciler creates a new ConfigMap if needed
func SecretReconciler(c reconcilers.Config, r *ContextReconciler) reconcilers.SubReconciler {
	c.Log = c.Log.WithName("ConfigMap")
	return &reconcilers.ChildReconciler{
		Config:        c,
		ParentType:    &api.Microservice{},
		ChildType:     &corev1.Secret{},
		ChildListType: &corev1.SecretList{},

		DesiredChild: func(micro *api.Microservice) (*corev1.Secret, error) {
			return &r.secret, nil
		},

		ReflectChildStatusOnParent: func(micro *api.Microservice, child *corev1.Secret, err error) {
			return
		},

		MergeBeforeUpdate: func(current, desired *corev1.Secret) {
			current.Labels = desired.Labels
		},

		SemanticEquals: func(a1, a2 *corev1.Secret) bool {
			return equality.Semantic.DeepEqual(a1.Labels, a2.Labels) && equality.Semantic.DeepEqual(a1.Data, a2.Data)
		},

		IndexField: ".metadata.serviceBindingSecretp",

		Sanitize: func(child *corev1.Secret) interface{} {
			return child.Data
		},
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
			bindingsToApply := findBindings(c, micro)
			trackBindings(c, bindingsToApply)
			updatedBindings := updateBindings(c, micro, bindingsToApply)
			return createDeployment(updatedBindings, micro), nil
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

		Setup: func(mgr reconcilers.Manager, bldr *reconcilers.Builder) error {
			bldr.Watches(&source.Kind{Type: &api.ServiceBinding{}}, reconcilers.EnqueueTracked(&api.ServiceBinding{}, c.Tracker, c.Scheme))
			return nil
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

func updatePodTemplate(template *corev1.PodTemplateSpec, bindings []api.ServiceBinding, source *api.Microservice) *corev1.PodTemplateSpec {
	micro := source.DeepCopy()
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

func findBindingsToApply(micro api.Microservice, bindingsMap map[string]api.ServiceBinding) []api.ServiceBinding {
	var bindingsToApply = []api.ServiceBinding{}
	for _, name := range micro.Spec.Bindings {
		if binding, ok := bindingsMap[name]; ok {
			bindingsToApply = append(bindingsToApply, binding)
		}
	}
	return bindingsToApply
}

func trackBindings(c reconcilers.Config, bindingsToApply []api.ServiceBinding) {
	for _, binding := range bindingsToApply {
		key := types.NamespacedName{
			Namespace: binding.Namespace,
			Name:      binding.Name,
		}
		c.Tracker.Track(
			tracker.NewKey(corev1.SchemeGroupVersion.WithKind("ServiceBinding"), key),
			types.NamespacedName{Namespace: binding.Namespace, Name: binding.Name},
		)
	}
}

func findBindings(c reconcilers.Config, micro *api.Microservice) []api.ServiceBinding {
	var bindingsToApply []api.ServiceBinding
	if len(micro.Spec.Bindings) > 0 {
		ctx := context.Background()
		var bindings api.ServiceBindingList
		if err := c.List(ctx, &bindings, &client.ListOptions{Namespace: corev1.NamespaceAll}); err != nil {
			c.Log.Error(err, "Unable to list Bindings")
			// Not fatal
		}
		bindingsMap := map[string]api.ServiceBinding{}
		for _, binding := range bindings.Items {
			bindingsMap[fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)] = binding
			if binding.Namespace == micro.Namespace {
				bindingsMap[binding.Name] = binding
			}
		}
		bindingsToApply = findBindingsToApply(*micro, bindingsMap)
	}
	return bindingsToApply
}

func updateBindings(c reconcilers.Config, micro *api.Microservice, bindings []api.ServiceBinding) []api.ServiceBinding {
	if len(bindings) == 0 {
		return bindings
	}
	var bindingsToApply = []api.ServiceBinding{}
	for _, binding := range bindings {
		bindingToApply := binding
		if binding.Namespace != micro.Namespace {
			bindingToApply = *bindingToApply.DeepCopy()
			for _, volume := range bindingToApply.Spec.Template.Spec.Volumes {
				if volume.ConfigMap != nil {
					volume.ConfigMap.Name = fmt.Sprintf("%s-%s", micro.Name, volume.ConfigMap.Name)
				}
				if volume.Secret != nil {
					volume.Secret.SecretName = fmt.Sprintf("%s-%s", micro.Name, volume.Secret.SecretName)
				}
			}
		}
		bindingsToApply = append(bindingsToApply, bindingToApply)
	}
	return bindingsToApply
}
