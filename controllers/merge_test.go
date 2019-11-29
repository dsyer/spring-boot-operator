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
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestMergeContainersWithEnv(t *testing.T) {
	source := corev1.Container{
		Image: "springguides/demo",
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "EXT_LIBS",
				Value: "/app/ext",
			},
		},
	}
	target := corev1.Container{}
	mergeContainers(source, &target)
	if target.Image != "springguides/demo" {
		t.Errorf("Container.Image = %s; want 'springguides/demo'", target.Image)
	}
	value := findEnvByName(target.Env, "EXT_LIBS")
	if value.Value != "/app/ext" {
		t.Errorf("Env['EXT_LIBS'] = %s; want '/app/ext'", value)
	}
}

func TestMergeContainersWithProbes(t *testing.T) {
	source := corev1.Container{
		Image: "springguides/demo",
		LivenessProbe: &corev1.Probe{
			InitialDelaySeconds: 30,
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{Path: "/info"},
			},
		},
	}
	target := corev1.Container{}
	mergeContainers(source, &target)
	if target.Image != "springguides/demo" {
		t.Errorf("Container.Image = %s; want 'springguides/demo'", target.Image)
	}
	if target.LivenessProbe.InitialDelaySeconds != 30 {
		t.Errorf("Container.LivenessProbe.InitialDelaySeconds = %d; want 30", target.LivenessProbe.InitialDelaySeconds)
		t.FailNow()
	}
	if target.LivenessProbe.Handler.HTTPGet == nil {
		t.Errorf("Container.LivenessProbe = %s; want not 'nil'", target.LivenessProbe)
		t.FailNow()
	}
	if target.LivenessProbe.Handler.HTTPGet.Path != "/info" {
		t.Errorf("Container.LivenessProbe.Handler.HTTPGet.Path = %s; want 'nil'", target.LivenessProbe.Handler.HTTPGet.Path)
	}
}
