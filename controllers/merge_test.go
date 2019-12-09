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

var (
	emptyContainer   = corev1.Container{Name: "__empty"}
	emptyEnvVar      = corev1.EnvVar{Name: "__empty"}
	emptyVolume      = corev1.Volume{Name: "__empty"}
	emptyVolumeMount = corev1.VolumeMount{Name: "__empty"}
)

func findVolumeByName(volumes []corev1.Volume, name string) *corev1.Volume {
	for index, volume := range volumes {
		if volume.Name == name {
			return &volumes[index]
		}
	}
	return &emptyVolume
}

func findVolumeMountByName(volumes []corev1.VolumeMount, name string) *corev1.VolumeMount {
	for index, volume := range volumes {
		if volume.Name == name {
			return &volumes[index]
		}
	}
	return &emptyVolumeMount
}

func findEnvByName(env []corev1.EnvVar, name string) *corev1.EnvVar {
	for index, v := range env {
		if v.Name == name {
			return &env[index]
		}
	}
	return &emptyEnvVar
}

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
	mergeResources(source, &target)
	if target.Image != "springguides/demo" {
		t.Errorf("Container.Image = %s; want 'springguides/demo'", target.Image)
	}
	value := findEnvByName(target.Env, "EXT_LIBS")
	if value.Value != "/app/ext" {
		t.Errorf("Env['EXT_LIBS'] = %s; want '/app/ext'", value)
	}
}

func TestMergeContainersWithCommand(t *testing.T) {
	source := corev1.Container{
		Image:   "springguides/demo",
		Command: []string{"foo", "bar"},
	}
	target := corev1.Container{}
	mergeResources(source, &target)
	if len(target.Command) != 2 {
		t.Errorf("Container.Command = %d; want 2", len(target.Command))
	}
	if target.Command[0] != "foo" {
		t.Errorf("Container.Command[0] = %s; want 'foo'", target.Command[0])
	}
}

func TestMergeContainersWithArgs(t *testing.T) {
	source := corev1.Container{
		Image: "springguides/demo",
		Args:  []string{"foo", "bar"},
	}
	target := corev1.Container{}
	mergeResources(source, &target)
	if len(target.Args) != 2 {
		t.Errorf("Container.Args = %d; want 2", len(target.Args))
	}
	if target.Args[0] != "foo" {
		t.Errorf("Container.Args[0] = %s; want 'foo'", target.Args[0])
	}
}

func TestMergeContainersWithWorkingDir(t *testing.T) {
	source := corev1.Container{
		Image:      "springguides/demo",
		WorkingDir: "/app",
	}
	target := corev1.Container{}
	mergeResources(source, &target)
	if target.WorkingDir != "/app" {
		t.Errorf("Container.WorkingDir = %s; want '/app'", target.WorkingDir)
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
	mergeResources(source, &target)
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
