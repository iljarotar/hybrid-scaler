package v1

import (
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
	// TODO: implement
}

func (in *HybridScalerStatus) DeepCopyInto(out *HybridScalerStatus) {
	// TODO: implement
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
