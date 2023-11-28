/*
Copyright 2023.

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
	"k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HybridScalerSpec defines the desired state of HybridScaler
type HybridScalerSpec struct {
	ScaleTargetRef v2.CrossVersionObjectReference `json:"scaleTargetRef"`
	MinReplicas    *int32                         `json:"minReplicas"`
	MaxReplicas    *int32                         `json:"maxReplicas"`

	// ResourcePolicy must be applied on a pod level
	ResourcePolicy ResourcePolicy `json:"resourcePolicy"`
}

type ResourcePolicy struct {
	MinAllowed          corev1.ResourceList           `json:"minAllowed"`
	MaxAllowed          corev1.ResourceList           `json:"maxAllowed"`
	TargetUtilization   map[corev1.ResourceName]int32 `json:"targetUtilization"`
	ControlledResources *[]corev1.ResourceName        `json:"controlledResources"`
}

type ContainerResources struct {
	Requests corev1.ResourceList `json:"requests"`
	Limits   corev1.ResourceList `json:"limits"`
}

// HybridScalerStatus defines the observed state of HybridScaler
type HybridScalerStatus struct {
	Replicas           int32                         `json:"replicas"`
	ContainerResources map[string]ContainerResources `json:"containerResources"`
	ContainerMetrics   []v1beta1.ContainerMetrics    `json:"containerMetrics"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HybridScaler is the Schema for the hybridscalers API
type HybridScaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HybridScalerSpec   `json:"spec,omitempty"`
	Status HybridScalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HybridScalerList contains a list of HybridScaler
type HybridScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HybridScaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HybridScaler{}, &HybridScalerList{})
}
