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
)

var (
	emptyContainer   = corev1.Container{Name: "__empty"}
	emptyEnvVar      = corev1.EnvVar{Name: "__empty"}
	emptyVolume      = corev1.Volume{Name: "__empty"}
	emptyVolumeMount = corev1.VolumeMount{Name: "__empty"}
)

func mergePodTemplates(source corev1.PodTemplateSpec, target *corev1.PodTemplateSpec) error {
	return Merge(source, target)
}

func findContainerByName(containers []corev1.Container, name string) *corev1.Container {
	for index, container := range containers {
		if container.Name == name {
			return &containers[index]
		}
	}
	return &emptyContainer
}

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

func mergeVolumes(source corev1.Volume, target *corev1.Volume) {
	Merge(source, target)
}

func mergeContainers(source corev1.Container, target *corev1.Container) {
	Merge(source, target)
}

func mergeProbes(source corev1.Probe, target *corev1.Probe) {
	Merge(source, target)
}

func mergeVolumeMounts(source corev1.VolumeMount, target *corev1.VolumeMount) {
	Merge(source, target)
}
