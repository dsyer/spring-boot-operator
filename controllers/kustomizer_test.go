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
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMergePod(t *testing.T) {
	target := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						Name: "foo",
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "BAR",
								Value: "foo",
							},
						},
					},
				},
			},
		}
	source := corev1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{},
			},
		}
	err := Merge(&target, source)
	if err != nil {
		t.Errorf("Failed to make resource map: %s", err)
		t.FailNow()
	}
	if target.ObjectMeta.Annotations["foo"] != "bar" {
		t.Errorf("Failed to merge annotations: %s", target.ObjectMeta.Annotations)
	}
	if len(target.Spec.Containers) != 1 {
		t.Errorf("Failed to merge containers expected 1 container found: %d", len(target.Spec.Containers))
	}
}
