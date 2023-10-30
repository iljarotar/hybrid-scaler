package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HybridScaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HybridScalerSpec   `json:"spec"`
	Status            HybridScalerStatus `json:"status"`
}

type HybridScalerSpec struct {
	// TODO: add fields
}

type HybridScalerStatus struct {
	// TODO: add fields
}

type HybridScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HybridScaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HybridScaler{}, &HybridScalerList{})
}
