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

// ServiceBindingSpec defines the desired state of ServiceBinding
type ServiceBindingSpec struct {
	Template corev1.PodTemplateSpec `json:"template,omitempty"`
}

// ServiceBindingStatus defines the observed state of ServiceBinding
type ServiceBindingStatus struct {
	Bound []string `json:"bound,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceBinding is the Schema for the springs API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Bound",type="string",JSONPath=".status.bound",description="bound microservices"
type ServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceBindingSpec   `json:"spec,omitempty"`
	Status ServiceBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceBindingList contains a list of Spring
type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceBinding{}, &ServiceBindingList{})
}
