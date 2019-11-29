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
	emptyProbe       = corev1.Probe{}
	emptyVolume      = corev1.Volume{Name: "__empty"}
	emptyVolumeMount = corev1.VolumeMount{Name: "__empty"}
)

func mergePodTemplates(source corev1.PodTemplateSpec, target *corev1.PodTemplateSpec) error {
	for k, v := range source.Annotations {
		if target.Annotations[k] == "" {
			if target.Annotations == nil {
				target.Annotations = map[string]string{}
			}
			// Only set if not already specified
			target.Annotations[k] = v
		}
	}
	// TODO: Copy more properties. DeepCopy is too blunt.
	if source.Spec.RestartPolicy != "" {
		target.Spec.RestartPolicy = source.Spec.RestartPolicy
	}
	for _, s := range source.Spec.Containers {
		t := findContainerByName(target.Spec.Containers, s.Name)
		if t.Name != emptyContainer.Name {
			mergeContainers(s, t)
		} else {
			target.Spec.Containers = append(target.Spec.Containers, s)
		}
	}
	for _, s := range source.Spec.InitContainers {
		t := findContainerByName(target.Spec.InitContainers, s.Name)
		if t.Name != emptyContainer.Name {
			mergeContainers(s, t)
		} else {
			target.Spec.InitContainers = append(target.Spec.InitContainers, s)
		}
	}
	for _, s := range source.Spec.Volumes {
		t := findVolumeByName(target.Spec.Volumes, s.Name)
		if t.Name != emptyVolume.Name {
			mergeVolumes(s, t)
		} else {
			target.Spec.Volumes = append(target.Spec.Volumes, s)
		}
	}
	return nil
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
	if source.ConfigMap != emptyVolume.ConfigMap {
		target.ConfigMap = source.ConfigMap
	}
	if source.Secret != emptyVolume.Secret {
		target.Secret = source.Secret
	}
	if source.HostPath != emptyVolume.HostPath {
		target.HostPath = source.HostPath
	}
}

func mergeContainers(source corev1.Container, target *corev1.Container) {
	if source.Image != emptyContainer.Image {
		target.Image = source.Image
	}
	for _, s := range source.VolumeMounts {
		t := findVolumeMountByName(target.VolumeMounts, s.Name)
		if t.Name != emptyVolumeMount.Name {
			mergeVolumeMounts(s, t)
		} else {
			target.VolumeMounts = append(target.VolumeMounts, s)
		}

	}
	for _, s := range source.Env {
		t := findEnvByName(target.Env, s.Name)
		if t.Name != emptyEnvVar.Name {
			t.Value = s.Value
		} else {
			target.Env = append(target.Env, s)
		}

	}
	if source.LivenessProbe != nil {
		if target.LivenessProbe == nil {
			target.LivenessProbe = source.LivenessProbe
		} else {
			mergeProbes(*source.LivenessProbe, target.LivenessProbe)
		}
	}
	if source.ReadinessProbe != nil {
		if target.ReadinessProbe == nil {
			target.ReadinessProbe = source.ReadinessProbe
		} else {
			mergeProbes(*source.ReadinessProbe, target.ReadinessProbe)
		}
	}
}

func mergeProbes(source corev1.Probe, target *corev1.Probe) {
	source.DeepCopyInto(target)
}

func mergeVolumeMounts(source corev1.VolumeMount, target *corev1.VolumeMount) {
	// TODO: something
}
