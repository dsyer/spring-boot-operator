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
	api "github.com/dsyer/sample-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func TestCreateService(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
		},
	}
	service := createService(&micro)
	if service.Name != "demo" {
		t.Errorf("Service.Name = %s; want 'demo'", service.Name)
	}
	if service.Namespace != "test" {
		t.Errorf("Service.Namespace = %s; want 'test'", service.Namespace)
	}
	if service.Labels["app"] != "demo" {
		t.Errorf("Service.Labels['app'] = %s; want 'demo'", service.Labels["app"])
	}
	if service.Spec.Selector["app"] != "demo" {
		t.Errorf("Service.Spec.Selector['app'] = %s; want 'demo'", service.Spec.Selector["app"])
	}
	if len(service.Spec.Ports) != 1 {
		t.Errorf("len(Service.Spec.Ports) = %d; want 1", len(service.Spec.Ports))
	}
	port := service.Spec.Ports[0]
	if port.TargetPort.IntVal != 8080 {
		t.Errorf("port.TargetPort = %d; want 8080", port.TargetPort.IntVal)
	}
	if port.Port != 80 {
		t.Errorf("port.Port = %d; want 80", port.Port)
	}
}

func TestCreateDeploymentVanilla(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	if deployment.Name != "demo" {
		t.Errorf("Deployment.Name = %s; want 'demo'", deployment.Name)
	}
	if deployment.Labels["app"] != "demo" {
		t.Errorf("Deployment.Labels['app'] = %s; want 'demo'", deployment.Labels["app"])
	}
	if len(deployment.Spec.Selector.MatchLabels) != 1 {
		t.Errorf("len(deployment.Spec.Selector.MatchLabels) = %d; want 1", len(deployment.Spec.Selector.MatchLabels))
	}
	if len(deployment.Spec.Template.ObjectMeta.Labels) != 1 {
		t.Errorf("len(deployment.Spec.Template.ObjectMeta.Labels) = %d; want 1", len(deployment.Spec.Template.ObjectMeta.Labels))
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("len(Containers) = %d; want 1", len(deployment.Spec.Template.Spec.Containers))
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != "springguides/demo" {
		t.Errorf("Container.Image = %s; want 'springguides/demo'", container.Image)
	}
	if container.LivenessProbe != nil {
		t.Errorf("Container.LivenessProbe = %s; want 'nil'", container.LivenessProbe)
	}

}

func TestCreateDeploymentActuators(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Actuators: true,
			Image:     "springguides/demo",
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.LivenessProbe == nil {
		t.Errorf("Container.LivenessProbe = %s; want not nil", container.LivenessProbe)
	}
	if container.ReadinessProbe == nil {
		t.Errorf("Container.ReadinessProbe = %s; want not nil", container.ReadinessProbe)
	}

}

func TestCreateDeploymentExistingAnonymousContainer(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name:  "FOO",
									Value: "BAR",
								},
							},
						},
					},
				},
			},
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("len(Containers) = %d; want 1", len(deployment.Spec.Template.Spec.Containers))
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != "springguides/demo" {
		t.Errorf("Container.Image = %s; want 'springguides/demo'", container.Image)
	}
	if container.LivenessProbe != nil {
		t.Errorf("Container.LivenessProbe = %s; want 'nil'", container.LivenessProbe)
	}
	if container.Env[0].Name != "FOO" {
		t.Errorf("Container.Env[0].Name = %s; want 'FOO'", container.Env[0].Name)
	}

}

func TestCreateDeploymentExistingContainer(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name: "app",
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name:  "FOO",
									Value: "BAR",
								},
							},
						},
					},
				},
			},
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("len(Containers) = %d; want 1", len(deployment.Spec.Template.Spec.Containers))
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != "springguides/demo" {
		t.Errorf("Container.Image = %s; want 'springguides/demo'", container.Image)
	}
	if container.LivenessProbe != nil {
		t.Errorf("Container.LivenessProbe = %s; want 'nil'", container.LivenessProbe)
	}
	if container.Env[0].Name != "FOO" {
		t.Errorf("Container.Env[0].Name = %s; want 'FOO'", container.Env[0].Name)
	}

}
func TestCreateDeploymentBindings(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image:    "springguides/demo",
			Bindings: []string{"mysql", "redis"},
		},
	}
	deployment := createDeployment(findBindingsToApply(micro, []api.ServiceBinding{}), &micro)
	if len(deployment.Spec.Template.Spec.Volumes) != 5 {
		t.Errorf("len(container.VolumeMounts) = %d; want 5", len(deployment.Spec.Template.Spec.Volumes))
		t.FailNow()
	}
	volume := deployment.Spec.Template.Spec.Volumes[1]
	if volume.Name != "mysql-secret" {
		t.Errorf("Volumes[1].Name = %s; want 'mysql-secret'", volume.Name)
	}
	if volume.VolumeSource.Secret.SecretName != "mysql-secret" {
		t.Errorf("Volumes[1].Name = %s; want 'mysql-secret'", volume.VolumeSource.Secret.SecretName)
	}
	volume = deployment.Spec.Template.Spec.Volumes[0]
	if volume.Name != "mysql-metadata" {
		t.Errorf("Volumes[0].Name = %s; want 'mysql-metadata'", volume.Name)
	}
	if volume.ConfigMap.Name != "mysql-metadata" {
		t.Errorf("Volumes[0].Name = %s; want 'mysql-metadata'", volume.ConfigMap.Name)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if len(container.VolumeMounts) != 1 {
		t.Errorf("len(container.VolumeMounts) = %d; want 1", len(container.VolumeMounts))
		t.FailNow()
	}
	var env corev1.EnvVar
	for _, item := range container.Env {
		if item.Name == "SPRING_CONFIG_LOCATION" {
			env = item
			break
		}
	}
	if env.Name == "" {
		t.Errorf("container.Env should contain 'SPRING_CONFIG_LOCATION', but was %s", container.Env)
	}
	if !strings.Contains(env.Value, "classpath:/,") {
		t.Errorf("SPRING_CONFIG_LOCATION should contain classpath:/, found %s", env.Value)
	}
	if !strings.Contains(env.Value, "file:///etc/config/") {
		t.Errorf("SPRING_CONFIG_LOCATION should contain file:///etc/config/, found %s", env.Value)
	}
	container = deployment.Spec.Template.Spec.InitContainers[0]
	if len(container.VolumeMounts) != 5 {
		t.Errorf("len(container.VolumeMounts) = %d; want 5", len(container.VolumeMounts))
		t.FailNow()
	}
	mount := container.VolumeMounts[1]
	if mount.Name != "mysql-metadata" {
		t.Errorf("container.VolumeMounts[0].Name = %s; want 'mysql-metadata'", container.VolumeMounts[0].Name)
	}
	mount = container.VolumeMounts[3]
	if mount.Name != "redis-metadata" {
		t.Errorf("container.VolumeMounts[1].Name = %s; want 'redis-metadata'", container.VolumeMounts[1].Name)
	}
	mount = container.VolumeMounts[0]
	if mount.Name != "config" {
		t.Errorf("container.VolumeMounts[1].Name = %s; want 'config'", container.VolumeMounts[1].Name)
	}

}

func TestCreateDeploymentProfiles(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image:    "springguides/demo",
			Profiles: []string{"mysql", "redis"},
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	container := deployment.Spec.Template.Spec.Containers[0]
	var env corev1.EnvVar
	for _, item := range container.Env {
		if item.Name == "SPRING_PROFILES_ACTIVE" {
			env = item
			break
		}
	}
	if env.Name == "" {
		t.Errorf("container.Env should contain 'SPRING_PROFILES_ACTIVE', but was %s", container.Env)
	}
	if env.Value != "mysql,redis" {
		t.Errorf("SPRING_PROFILES_ACTIVE should contain 'mysql', found %s", env.Value)
	}

}

func TestCreateDeploymentAnnotations(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"foo": "bar"},
				},
			},
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	if deployment.Spec.Template.ObjectMeta.Annotations["foo"] != "bar" {
		t.Errorf("deployment.Spec.Template.ObjectMeta.Annotations['foo'] = %s; want 'bar'", deployment.Spec.Template.ObjectMeta.Annotations["foo"])
	}

}

func TestUpdateDeploymentProfiles(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	container := deployment.Spec.Template.Spec.Containers[0]
	if len(container.Env) > 0 {
		t.Errorf("container.Env should be empty but found %s", container.Env)
	}
	micro.Spec.Profiles = []string{"mysql", "redis"}
	updateDeployment(deployment, []api.ServiceBinding{}, &micro)
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("len(Containers) = %d; want 1", len(deployment.Spec.Template.Spec.Containers))
	}
	container = deployment.Spec.Template.Spec.Containers[0]
	var env corev1.EnvVar
	for _, item := range container.Env {
		if item.Name == "SPRING_PROFILES_ACTIVE" {
			env = item
			break
		}
	}
	if env.Name == "" {
		t.Errorf("container.Env should contain 'SPRING_PROFILES_ACTIVE', but was %s", container.Env)
	}
	if env.Value != "mysql,redis" {
		t.Errorf("SPRING_PROFILES_ACTIVE should contain 'mysql', found %s", env.Value)
	}

}

func TestUpdateImage(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != "springguides/demo" {
		t.Errorf("Container.Image = %s; want 'springguides/demo'", container.Image)
	}
	micro.Spec.Image = "springguides/demo:last"
	updateDeployment(deployment, []api.ServiceBinding{}, &micro)
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("len(Containers) = %d; want 1", len(deployment.Spec.Template.Spec.Containers))
	}
	container = deployment.Spec.Template.Spec.Containers[0]
	if container.Image != "springguides/demo:last" {
		t.Errorf("Container.Image = %s; want 'springguides/demo:last'", container.Image)
	}

}

func TestBindingAnnotations(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	if deployment.Spec.Template.ObjectMeta.Annotations["foo"] != "" {
		t.Errorf("deployment.Spec.Template.ObjectMeta.Annotations['foo'] = %s; want ''", deployment.Spec.Template.ObjectMeta.Annotations["foo"])
	}
	bindings := []api.ServiceBinding{
		api.ServiceBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysql",
			},
			Spec: api.ServiceBindingSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
		},
	}
	micro.Spec.Bindings = []string{"mysql"}
	updateDeployment(deployment, bindings, &micro)
	if deployment.Spec.Template.ObjectMeta.Annotations["foo"] != "bar" {
		t.Errorf("deployment.Spec.Template.ObjectMeta.Annotations['foo'] = %s; want 'bar'", deployment.Spec.Template.ObjectMeta.Annotations["foo"])
	}

}

func TestBindingVolumes(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
		},
	}
	bindings := []api.ServiceBinding{}
	deployment := createDeployment(bindings, &micro)
	if len(deployment.Spec.Template.Spec.Volumes) != 0 {
		t.Errorf("len(deployment.Spec.Template.Spec.Volumes) = %d; want 0", len(deployment.Spec.Template.Spec.Volumes))
	}
	micro.Spec.Bindings = []string{"mysql"}
	bindings = append(bindings, defaultBinding("mysql", micro))
	updateDeployment(deployment, bindings, &micro)
	if len(deployment.Spec.Template.Spec.Volumes) != 3 {
		t.Errorf("len(deployment.Spec.Template.Spec.Volumes) = %d; want 3", len(deployment.Spec.Template.Spec.Volumes))
	}
	micro.Spec.Bindings = []string{"mysql"}
	updateDeployment(deployment, bindings, &micro)
	if len(deployment.Spec.Template.Spec.Volumes) != 3 {
		t.Errorf("len(deployment.Spec.Template.Spec.Volumes) = %d; want 3", len(deployment.Spec.Template.Spec.Volumes))
	}

}

func TestDefaultBindingEnvVars(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Image: "springguides/demo",
		},
	}
	binding := defaultBinding("mysql", micro)
	if len(binding.Spec.Env) != 2 {
		t.Errorf("len(binding.Spec.Env) = %d; want 2", len(binding.Spec.Env))
	}
}

func TestBindingPod(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
	}
	deployment := createDeployment([]api.ServiceBinding{}, &micro)
	if deployment.Spec.Template.Spec.RestartPolicy != "" {
		t.Errorf("deployment.Spec.Template.Spec.RestartPolicy = %s; want ''", deployment.Spec.Template.Spec.RestartPolicy)
	}
	bindings := []api.ServiceBinding{
		api.ServiceBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysql",
			},
			Spec: api.ServiceBindingSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		},
	}
	micro.Spec.Bindings = []string{"mysql"}
	updateDeployment(deployment, bindings, &micro)
	if deployment.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("deployment.Spec.Template.Spec.RestartPolicy = %s; want 'Never'", deployment.Spec.Template.Spec.RestartPolicy)
	}

}

func TestBindingEnvVars(t *testing.T) {
	micro := api.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "test",
		},
		Spec: api.MicroserviceSpec{
			Bindings: []string{"mysql", "other"},
		},
	}
	bindings := []api.ServiceBinding{
		api.ServiceBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysql",
			},
			Spec: api.ServiceBindingSpec{
				Env: []api.EnvVar{
					api.EnvVar{Name: "FOO", Value: "bar"},
					api.EnvVar{Name: "SPLAT", Values: []string{"one", "two"}},
					api.EnvVar{Name: "BAR", Value: "spam"},
				},
			},
		},
		api.ServiceBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "other",
			},
			Spec: api.ServiceBindingSpec{
				Env: []api.EnvVar{
					api.EnvVar{Name: "SPLAT", Values: []string{"one", "three"}},
				},
			},
		},
	}
	deployment := createDeployment(bindings, &micro)
	container := deployment.Spec.Template.Spec.Containers[0]
	if findEnvByName(container.Env, "FOO").Value != "bar" {
		t.Errorf("container.Env[FOO] = %s; want 'bar'", container.Env)
	}
	if findEnvByName(container.Env, "BAR").Value != "spam" {
		t.Errorf("container.Env[BAR] = %s; want 'spam'", container.Env)
	}
	if findEnvByName(container.Env, "SPLAT").Value != "one,two,three" {
		t.Errorf("container.Env[SPLAT] = %s; want 'one,two,three'", container.Env)
	}
}

// mergeEnvVars
