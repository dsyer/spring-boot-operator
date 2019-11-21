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
	deployment := createDeployment(&micro)
	if deployment.Name != "demo" {
		t.Errorf("Deployment.Name = %s; want 'demo'", deployment.Name)
	}
	if deployment.Labels["app"] != "demo" {
		t.Errorf("Service.Labels['app'] = %s; want 'demo'", deployment.Labels["app"])
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
	deployment := createDeployment(&micro)
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.LivenessProbe == nil {
		t.Errorf("Container.LivenessProbe = %s; want not nil", container.LivenessProbe)
	}
	if container.ReadinessProbe == nil {
		t.Errorf("Container.ReadinessProbe = %s; want not nil", container.ReadinessProbe)
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
			Pod: corev1.PodSpec{
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
	}
	deployment := createDeployment(&micro)
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
