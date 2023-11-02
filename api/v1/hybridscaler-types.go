package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/autoscaling"
)

type HybridScaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HybridScalerSpec   `json:"spec"`
	Status            HybridScalerStatus `json:"status"`
}

type HybridScalerSpec struct {
	ScaleTargetRef autoscaling.CrossVersionObjectReference `json:"scaleTargetRef"`
	MinReplicas    *int32                                  `json:"minReplicas"`
	MaxReplicas    *int32                                  `json:"maxReplicas"`
	ResourcePolicy ResourcePolicy                          `json:"resourcePolicy"`
}

type ResourcePolicy struct {
	MinAllowed          corev1.ResourceList    `json:"minAllowed,omitempty" protobuf:"bytes,3,rep,name=minAllowed,casttype=ResourceList,castkey=ResourceName"`
	MaxAllowed          corev1.ResourceList    `json:"maxAllowed,omitempty" protobuf:"bytes,4,rep,name=maxAllowed,casttype=ResourceList,castkey=ResourceName"`
	ControlledResources *[]corev1.ResourceName `json:"controlledResources,omitempty" patchStrategy:"merge" protobuf:"bytes,5,rep,name=controlledResources"`
}

type HybridScalerStatus struct {
	CurrentReplicas   int32 `json:"currentReplicas"`
	CpuUtilization    int32 `json:"cpuUtilization"`
	MemoryUtilization int32 `json:"memoryUtilization"`
	RequestRate       int32 `json:"requestRate"`
	ResponseTime      int32 `json:"responseTime"`
}

type HybridScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HybridScaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HybridScaler{}, &HybridScalerList{})
}
