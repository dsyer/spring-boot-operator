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

// SpringSpec defines the desired state of Spring
type SpringSpec struct {
	Foo string `json:"foo,omitempty"`
}

// SpringStatus defines the observed state of Spring
type SpringStatus struct {
}

// +kubebuilder:object:root=true

// Spring is the Schema for the springs API
type Spring struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpringSpec   `json:"spec,omitempty"`
	Status SpringStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpringList contains a list of Spring
type SpringList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Spring `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Spring{}, &SpringList{})
}
