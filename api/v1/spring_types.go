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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MicroserviceSpec defines the desired state of Microservice
type MicroserviceSpec struct {
	Image     string                 `json:"image,omitempty"`
	Args      []string               `json:"args,omitempty"`
	Actuators bool                   `json:"actuators,omitempty"`
	Template  corev1.PodTemplateSpec `json:"template,omitempty"`
	Bindings  []string               `json:"bindings,omitempty"`
	Profiles  []string               `json:"profiles,omitempty"`
}

// MicroserviceStatus defines the observed state of Microservice
type MicroserviceStatus struct {
	ServiceName string `json:"serviceName,omitempty"`
	Label       string `json:"label,omitempty"`
	Running     bool   `json:"running,omitempty"`
}

// +kubebuilder:object:root=true

// Microservice is the Schema for the springs API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image",description="image label"
// +kubebuilder:printcolumn:name="Running",type="boolean",JSONPath=".status.running",description="deployment status"
type Microservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicroserviceSpec   `json:"spec,omitempty"`
	Status MicroserviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MicroserviceList contains a list of Spring
type MicroserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Microservice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Microservice{}, &MicroserviceList{})
}
