//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2020 The Knative Authors

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PipelineLoop) DeepCopyInto(out *PipelineLoop) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PipelineLoop.
func (in *PipelineLoop) DeepCopy() *PipelineLoop {
	if in == nil {
		return nil
	}
	out := new(PipelineLoop)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PipelineLoop) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PipelineLoopList) DeepCopyInto(out *PipelineLoopList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PipelineLoop, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PipelineLoopList.
func (in *PipelineLoopList) DeepCopy() *PipelineLoopList {
	if in == nil {
		return nil
	}
	out := new(PipelineLoopList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PipelineLoopList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PipelineLoopPipelineRunStatus) DeepCopyInto(out *PipelineLoopPipelineRunStatus) {
	*out = *in
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(tektonv1.PipelineRunStatus)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PipelineLoopPipelineRunStatus.
func (in *PipelineLoopPipelineRunStatus) DeepCopy() *PipelineLoopPipelineRunStatus {
	if in == nil {
		return nil
	}
	out := new(PipelineLoopPipelineRunStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PipelineLoopRunStatus) DeepCopyInto(out *PipelineLoopRunStatus) {
	*out = *in
	if in.PipelineLoopSpec != nil {
		in, out := &in.PipelineLoopSpec, &out.PipelineLoopSpec
		*out = new(PipelineLoopSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.PipelineRuns != nil {
		in, out := &in.PipelineRuns, &out.PipelineRuns
		*out = make(map[string]*PipelineLoopPipelineRunStatus, len(*in))
		for key, val := range *in {
			var outVal *PipelineLoopPipelineRunStatus
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(PipelineLoopPipelineRunStatus)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PipelineLoopRunStatus.
func (in *PipelineLoopRunStatus) DeepCopy() *PipelineLoopRunStatus {
	if in == nil {
		return nil
	}
	out := new(PipelineLoopRunStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PipelineLoopSpec) DeepCopyInto(out *PipelineLoopSpec) {
	*out = *in
	if in.PipelineRef != nil {
		in, out := &in.PipelineRef, &out.PipelineRef
		*out = new(tektonv1.PipelineRef)
		**out = **in
	}
	if in.PipelineSpec != nil {
		in, out := &in.PipelineSpec, &out.PipelineSpec
		*out = new(tektonv1.PipelineSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(v1.Duration)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PipelineLoopSpec.
func (in *PipelineLoopSpec) DeepCopy() *PipelineLoopSpec {
	if in == nil {
		return nil
	}
	out := new(PipelineLoopSpec)
	in.DeepCopyInto(out)
	return out
}
