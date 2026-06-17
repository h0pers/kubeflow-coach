/*
Copyright 2026.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterTrainingRuntimeSpec defines the infrastructure blueprint for training jobs.
type ClusterTrainingRuntimeSpec struct {
	// +optional
	// +kubebuilder:default=1
	NumNodes *int32 `json:"numNodes,omitempty"`

	// +required
	Template corev1.PodTemplateSpec `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterTrainingRuntime is a cluster-scoped blueprint that defines
// the infrastructure template for TrainJobs.
type ClusterTrainingRuntime struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ClusterTrainingRuntimeSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// ClusterTrainingRuntimeList contains a list of ClusterTrainingRuntime.
type ClusterTrainingRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ClusterTrainingRuntime `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ClusterTrainingRuntime{}, &ClusterTrainingRuntimeList{})
		return nil
	})
}
