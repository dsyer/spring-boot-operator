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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpringApplicationSpec defines the desired state of SpringApplication
type SpringApplicationSpec struct {
	Image string `json:"image,omitempty"`
}

// SpringApplicationStatus defines the observed state of SpringApplication
type SpringApplicationStatus struct {
	ServiceName string `json:"serviceName,omitempty"`
	Label       string `json:"label,omitempty"`
	Running     bool   `json:"running,omitempty"`
}

// +kubebuilder:object:root=true

// SpringApplication is the Schema for the springs API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image",description="image label"
// +kubebuilder:printcolumn:name="Running",type="boolean",JSONPath=".status.running",description="deployment status"
type SpringApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpringApplicationSpec   `json:"spec,omitempty"`
	Status SpringApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpringApplicationList contains a list of Spring
type SpringApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpringApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpringApplication{}, &SpringApplicationList{})
}
