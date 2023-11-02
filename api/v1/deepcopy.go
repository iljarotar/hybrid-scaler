package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (in *HybridScaler) DeepCopyInto(out *HybridScaler) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *HybridScalerSpec) DeepCopyInto(out *HybridScalerSpec) {
	*out = *in
	in.ScaleTargetRef.DeepCopyInto(&out.ScaleTargetRef)
	in.ResourcePolicy.DeepCopyInto(&out.ResourcePolicy)
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	if in.MaxReplicas != nil {
		in, out := &in.MaxReplicas, &out.MaxReplicas
		*out = new(int32)
		**out = **in
	}
}

func (in *ResourcePolicy) DeepCopyInto(out *ResourcePolicy) {
	*out = *in
	if in.MinAllowed != nil {
		in, out := &in.MinAllowed, &out.MinAllowed
		*out = make(corev1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
	if in.MaxAllowed != nil {
		in, out := &in.MaxAllowed, &out.MaxAllowed
		*out = make(corev1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
	if in.ControlledResources != nil {
		in, out := &in.ControlledResources, &out.ControlledResources
		*out = new([]corev1.ResourceName)
		if **in != nil {
			in, out := *in, *out
			*out = make([]corev1.ResourceName, len(*in))
			copy(*out, *in)
		}
	}
}

func (in *HybridScalerStatus) DeepCopyInto(out *HybridScalerStatus) {
	*out = *in
}

func (in *HybridScalerList) DeepCopyInto(out *HybridScalerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HybridScaler, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *HybridScaler) DeepCopy() *HybridScaler {
	if in == nil {
		return nil
	}
	out := new(HybridScaler)
	in.DeepCopyInto(out)
	return out
}

func (in *HybridScalerSpec) DeepCopy() *HybridScalerSpec {
	if in == nil {
		return nil
	}
	out := new(HybridScalerSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourcePolicy) DeepCopy() *ResourcePolicy {
	if in == nil {
		return nil
	}
	out := new(ResourcePolicy)
	in.DeepCopyInto(out)
	return out
}

func (in *HybridScalerStatus) DeepCopy() *HybridScalerStatus {
	if in == nil {
		return nil
	}
	out := new(HybridScalerStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *HybridScalerList) DeepCopy() *HybridScalerList {
	if in == nil {
		return nil
	}
	out := new(HybridScalerList)
	in.DeepCopyInto(out)
	return out
}

func (in *HybridScaler) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *HybridScalerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
