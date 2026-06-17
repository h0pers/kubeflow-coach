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

const (
	TrainJobCreated   = "Created"
	TrainJobComplete  = "Complete"
	TrainJobFailed    = "Failed"
	TrainJobSuspended = "Suspended"
)

// RuntimeRef references a ClusterTrainingRuntime by name.
type RuntimeRef struct {
	// +required
	Name string `json:"name"`
}

// TrainJobSpec defines the desired state of TrainJob.
type TrainJobSpec struct {
	// +required
	RuntimeRef RuntimeRef `json:"runtimeRef"`

	// +optional
	Image *string `json:"image,omitempty"`

	// +optional
	Command []string `json:"command,omitempty"`

	// +optional
	Args []string `json:"args,omitempty"`

	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// TrainJobStatus defines the observed state of TrainJob.
type TrainJobStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	Active *int32 `json:"active,omitempty"`

	// +optional
	Succeeded *int32 `json:"succeeded,omitempty"`

	// +optional
	Failed *int32 `json:"failed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TrainJob is the Schema for the trainjobs API.
type TrainJob struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec TrainJobSpec `json:"spec"`

	// +optional
	Status TrainJobStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TrainJobList contains a list of TrainJob.
type TrainJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TrainJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &TrainJob{}, &TrainJobList{})
		return nil
	})
}
