// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: backend/api/v1beta1/filter.proto

package go_client

import (
	context "context"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Op is the operation to apply.
type Predicate_Op int32

const (
	Predicate_UNKNOWN Predicate_Op = 0
	// Operators on scalar values. Only applies to one of |int_value|,
	// |long_value|, |string_value| or |timestamp_value|.
	Predicate_EQUALS              Predicate_Op = 1
	Predicate_NOT_EQUALS          Predicate_Op = 2
	Predicate_GREATER_THAN        Predicate_Op = 3
	Predicate_GREATER_THAN_EQUALS Predicate_Op = 5
	Predicate_LESS_THAN           Predicate_Op = 6
	Predicate_LESS_THAN_EQUALS    Predicate_Op = 7
	// Checks if the value is a member of a given array, which should be one of
	// |int_values|, |long_values| or |string_values|.
	Predicate_IN Predicate_Op = 8
	// Checks if the value contains |string_value| as a substring match. Only
	// applies to |string_value|.
	Predicate_IS_SUBSTRING Predicate_Op = 9
)

// Enum value maps for Predicate_Op.
var (
	Predicate_Op_name = map[int32]string{
		0: "UNKNOWN",
		1: "EQUALS",
		2: "NOT_EQUALS",
		3: "GREATER_THAN",
		5: "GREATER_THAN_EQUALS",
		6: "LESS_THAN",
		7: "LESS_THAN_EQUALS",
		8: "IN",
		9: "IS_SUBSTRING",
	}
	Predicate_Op_value = map[string]int32{
		"UNKNOWN":             0,
		"EQUALS":              1,
		"NOT_EQUALS":          2,
		"GREATER_THAN":        3,
		"GREATER_THAN_EQUALS": 5,
		"LESS_THAN":           6,
		"LESS_THAN_EQUALS":    7,
		"IN":                  8,
		"IS_SUBSTRING":        9,
	}
)

func (x Predicate_Op) Enum() *Predicate_Op {
	p := new(Predicate_Op)
	*p = x
	return p
}

func (x Predicate_Op) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Predicate_Op) Descriptor() protoreflect.EnumDescriptor {
	return file_backend_api_v1beta1_filter_proto_enumTypes[0].Descriptor()
}

func (Predicate_Op) Type() protoreflect.EnumType {
	return &file_backend_api_v1beta1_filter_proto_enumTypes[0]
}

func (x Predicate_Op) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Predicate_Op.Descriptor instead.
func (Predicate_Op) EnumDescriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_filter_proto_rawDescGZIP(), []int{0, 0}
}

// Predicate captures individual conditions that must be true for a resource
// being filtered.
type Predicate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Op  Predicate_Op `protobuf:"varint,1,opt,name=op,proto3,enum=v1beta1.Predicate_Op" json:"op,omitempty"`
	Key string       `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	// Types that are assignable to Value:
	//	*Predicate_IntValue
	//	*Predicate_LongValue
	//	*Predicate_StringValue
	//	*Predicate_TimestampValue
	//	*Predicate_IntValues
	//	*Predicate_LongValues
	//	*Predicate_StringValues
	Value isPredicate_Value `protobuf_oneof:"value"`
}

func (x *Predicate) Reset() {
	*x = Predicate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v1beta1_filter_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Predicate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Predicate) ProtoMessage() {}

func (x *Predicate) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v1beta1_filter_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Predicate.ProtoReflect.Descriptor instead.
func (*Predicate) Descriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_filter_proto_rawDescGZIP(), []int{0}
}

func (x *Predicate) GetOp() Predicate_Op {
	if x != nil {
		return x.Op
	}
	return Predicate_UNKNOWN
}

func (x *Predicate) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (m *Predicate) GetValue() isPredicate_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (x *Predicate) GetIntValue() int32 {
	if x, ok := x.GetValue().(*Predicate_IntValue); ok {
		return x.IntValue
	}
	return 0
}

func (x *Predicate) GetLongValue() int64 {
	if x, ok := x.GetValue().(*Predicate_LongValue); ok {
		return x.LongValue
	}
	return 0
}

func (x *Predicate) GetStringValue() string {
	if x, ok := x.GetValue().(*Predicate_StringValue); ok {
		return x.StringValue
	}
	return ""
}

func (x *Predicate) GetTimestampValue() *timestamppb.Timestamp {
	if x, ok := x.GetValue().(*Predicate_TimestampValue); ok {
		return x.TimestampValue
	}
	return nil
}

func (x *Predicate) GetIntValues() *IntValues {
	if x, ok := x.GetValue().(*Predicate_IntValues); ok {
		return x.IntValues
	}
	return nil
}

func (x *Predicate) GetLongValues() *LongValues {
	if x, ok := x.GetValue().(*Predicate_LongValues); ok {
		return x.LongValues
	}
	return nil
}

func (x *Predicate) GetStringValues() *StringValues {
	if x, ok := x.GetValue().(*Predicate_StringValues); ok {
		return x.StringValues
	}
	return nil
}

type isPredicate_Value interface {
	isPredicate_Value()
}

type Predicate_IntValue struct {
	IntValue int32 `protobuf:"varint,3,opt,name=int_value,json=intValue,proto3,oneof"`
}

type Predicate_LongValue struct {
	LongValue int64 `protobuf:"varint,4,opt,name=long_value,json=longValue,proto3,oneof"`
}

type Predicate_StringValue struct {
	StringValue string `protobuf:"bytes,5,opt,name=string_value,json=stringValue,proto3,oneof"`
}

type Predicate_TimestampValue struct {
	// Timestamp values will be converted to Unix time (seconds since the epoch)
	// prior to being used in a filtering operation.
	TimestampValue *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=timestamp_value,json=timestampValue,proto3,oneof"`
}

type Predicate_IntValues struct {
	// Array values below are only meant to be used by the IN operator.
	IntValues *IntValues `protobuf:"bytes,7,opt,name=int_values,json=intValues,proto3,oneof"`
}

type Predicate_LongValues struct {
	LongValues *LongValues `protobuf:"bytes,8,opt,name=long_values,json=longValues,proto3,oneof"`
}

type Predicate_StringValues struct {
	StringValues *StringValues `protobuf:"bytes,9,opt,name=string_values,json=stringValues,proto3,oneof"`
}

func (*Predicate_IntValue) isPredicate_Value() {}

func (*Predicate_LongValue) isPredicate_Value() {}

func (*Predicate_StringValue) isPredicate_Value() {}

func (*Predicate_TimestampValue) isPredicate_Value() {}

func (*Predicate_IntValues) isPredicate_Value() {}

func (*Predicate_LongValues) isPredicate_Value() {}

func (*Predicate_StringValues) isPredicate_Value() {}

type IntValues struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Values []int32 `protobuf:"varint,1,rep,packed,name=values,proto3" json:"values,omitempty"`
}

func (x *IntValues) Reset() {
	*x = IntValues{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v1beta1_filter_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IntValues) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IntValues) ProtoMessage() {}

func (x *IntValues) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v1beta1_filter_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IntValues.ProtoReflect.Descriptor instead.
func (*IntValues) Descriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_filter_proto_rawDescGZIP(), []int{1}
}

func (x *IntValues) GetValues() []int32 {
	if x != nil {
		return x.Values
	}
	return nil
}

type StringValues struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Values []string `protobuf:"bytes,2,rep,name=values,proto3" json:"values,omitempty"`
}

func (x *StringValues) Reset() {
	*x = StringValues{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v1beta1_filter_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StringValues) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StringValues) ProtoMessage() {}

func (x *StringValues) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v1beta1_filter_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StringValues.ProtoReflect.Descriptor instead.
func (*StringValues) Descriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_filter_proto_rawDescGZIP(), []int{2}
}

func (x *StringValues) GetValues() []string {
	if x != nil {
		return x.Values
	}
	return nil
}

type LongValues struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Values []int64 `protobuf:"varint,3,rep,packed,name=values,proto3" json:"values,omitempty"`
}

func (x *LongValues) Reset() {
	*x = LongValues{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v1beta1_filter_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LongValues) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LongValues) ProtoMessage() {}

func (x *LongValues) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v1beta1_filter_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LongValues.ProtoReflect.Descriptor instead.
func (*LongValues) Descriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_filter_proto_rawDescGZIP(), []int{3}
}

func (x *LongValues) GetValues() []int64 {
	if x != nil {
		return x.Values
	}
	return nil
}

// Filter is used to filter resources returned from a ListXXX request.
//
// Example filters:
// 1) Filter runs with status = 'Running'
// filter {
//   predicate {
//     key: "status"
//     op: EQUALS
//     string_value: "Running"
//   }
// }
//
// 2) Filter runs that succeeded since Dec 1, 2018
// filter {
//   predicate {
//     key: "status"
//     op: EQUALS
//     string_value: "Succeeded"
//   }
//   predicate {
//     key: "created_at"
//     op: GREATER_THAN
//     timestamp_value {
//       seconds: 1543651200
//     }
//   }
// }
//
// 3) Filter runs with one of labels 'label_1' or 'label_2'
//
// filter {
//   predicate {
//     key: "label"
//     op: IN
//     string_values {
//       value: 'label_1'
//       value: 'label_2'
//     }
//   }
// }
type Filter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// All predicates are AND-ed when this filter is applied.
	Predicates []*Predicate `protobuf:"bytes,1,rep,name=predicates,proto3" json:"predicates,omitempty"`
}

func (x *Filter) Reset() {
	*x = Filter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v1beta1_filter_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Filter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Filter) ProtoMessage() {}

func (x *Filter) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v1beta1_filter_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Filter.ProtoReflect.Descriptor instead.
func (*Filter) Descriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_filter_proto_rawDescGZIP(), []int{4}
}

func (x *Filter) GetPredicates() []*Predicate {
	if x != nil {
		return x.Predicates
	}
	return nil
}

var File_backend_api_v1beta1_filter_proto protoreflect.FileDescriptor

var file_backend_api_v1beta1_filter_proto_rawDesc = []byte{
	0x0a, 0x20, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31,
	0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x07, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x1a, 0x1c, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbe, 0x04, 0x0a, 0x09, 0x50,
	0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x12, 0x25, 0x0a, 0x02, 0x6f, 0x70, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x15, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x50,
	0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x2e, 0x4f, 0x70, 0x52, 0x02, 0x6f, 0x70, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x1d, 0x0a, 0x09, 0x69, 0x6e, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x05, 0x48, 0x00, 0x52, 0x08, 0x69, 0x6e, 0x74, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x12, 0x1f, 0x0a, 0x0a, 0x6c, 0x6f, 0x6e, 0x67, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x03, 0x48, 0x00, 0x52, 0x09, 0x6c, 0x6f, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x12, 0x23, 0x0a, 0x0c, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x5f, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0b, 0x73, 0x74, 0x72, 0x69, 0x6e,
	0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x45, 0x0a, 0x0f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x48, 0x00, 0x52, 0x0e, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x33, 0x0a,
	0x0a, 0x69, 0x6e, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x12, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x49, 0x6e, 0x74, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x48, 0x00, 0x52, 0x09, 0x69, 0x6e, 0x74, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x73, 0x12, 0x36, 0x0a, 0x0b, 0x6c, 0x6f, 0x6e, 0x67, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x73, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61,
	0x31, 0x2e, 0x4c, 0x6f, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x48, 0x00, 0x52, 0x0a,
	0x6c, 0x6f, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x3c, 0x0a, 0x0d, 0x73, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x18, 0x09, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x15, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x53, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x48, 0x00, 0x52, 0x0c, 0x73, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x22, 0x97, 0x01, 0x0a, 0x02, 0x4f, 0x70, 0x12,
	0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06,
	0x45, 0x51, 0x55, 0x41, 0x4c, 0x53, 0x10, 0x01, 0x12, 0x0e, 0x0a, 0x0a, 0x4e, 0x4f, 0x54, 0x5f,
	0x45, 0x51, 0x55, 0x41, 0x4c, 0x53, 0x10, 0x02, 0x12, 0x10, 0x0a, 0x0c, 0x47, 0x52, 0x45, 0x41,
	0x54, 0x45, 0x52, 0x5f, 0x54, 0x48, 0x41, 0x4e, 0x10, 0x03, 0x12, 0x17, 0x0a, 0x13, 0x47, 0x52,
	0x45, 0x41, 0x54, 0x45, 0x52, 0x5f, 0x54, 0x48, 0x41, 0x4e, 0x5f, 0x45, 0x51, 0x55, 0x41, 0x4c,
	0x53, 0x10, 0x05, 0x12, 0x0d, 0x0a, 0x09, 0x4c, 0x45, 0x53, 0x53, 0x5f, 0x54, 0x48, 0x41, 0x4e,
	0x10, 0x06, 0x12, 0x14, 0x0a, 0x10, 0x4c, 0x45, 0x53, 0x53, 0x5f, 0x54, 0x48, 0x41, 0x4e, 0x5f,
	0x45, 0x51, 0x55, 0x41, 0x4c, 0x53, 0x10, 0x07, 0x12, 0x06, 0x0a, 0x02, 0x49, 0x4e, 0x10, 0x08,
	0x12, 0x10, 0x0a, 0x0c, 0x49, 0x53, 0x5f, 0x53, 0x55, 0x42, 0x53, 0x54, 0x52, 0x49, 0x4e, 0x47,
	0x10, 0x09, 0x42, 0x07, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x23, 0x0a, 0x09, 0x49,
	0x6e, 0x74, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x05, 0x52, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73,
	0x22, 0x26, 0x0a, 0x0c, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73,
	0x12, 0x16, 0x0a, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x22, 0x24, 0x0a, 0x0a, 0x4c, 0x6f, 0x6e, 0x67,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x03, 0x52, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x22, 0x3c,
	0x0a, 0x06, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x32, 0x0a, 0x0a, 0x70, 0x72, 0x65, 0x64,
	0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x76,
	0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x50, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65,
	0x52, 0x0a, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x32, 0x45, 0x0a, 0x12,
	0x44, 0x75, 0x6d, 0x6d, 0x79, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x12, 0x2f, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12,
	0x0f, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72,
	0x1a, 0x0f, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x46, 0x69, 0x6c, 0x74, 0x65,
	0x72, 0x22, 0x00, 0x42, 0x3d, 0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2f, 0x70, 0x69, 0x70, 0x65, 0x6c,
	0x69, 0x6e, 0x65, 0x73, 0x2f, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x67, 0x6f, 0x5f, 0x63, 0x6c, 0x69, 0x65,
	0x6e, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_backend_api_v1beta1_filter_proto_rawDescOnce sync.Once
	file_backend_api_v1beta1_filter_proto_rawDescData = file_backend_api_v1beta1_filter_proto_rawDesc
)

func file_backend_api_v1beta1_filter_proto_rawDescGZIP() []byte {
	file_backend_api_v1beta1_filter_proto_rawDescOnce.Do(func() {
		file_backend_api_v1beta1_filter_proto_rawDescData = protoimpl.X.CompressGZIP(file_backend_api_v1beta1_filter_proto_rawDescData)
	})
	return file_backend_api_v1beta1_filter_proto_rawDescData
}

var file_backend_api_v1beta1_filter_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_backend_api_v1beta1_filter_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_backend_api_v1beta1_filter_proto_goTypes = []interface{}{
	(Predicate_Op)(0),             // 0: v1beta1.Predicate.Op
	(*Predicate)(nil),             // 1: v1beta1.Predicate
	(*IntValues)(nil),             // 2: v1beta1.IntValues
	(*StringValues)(nil),          // 3: v1beta1.StringValues
	(*LongValues)(nil),            // 4: v1beta1.LongValues
	(*Filter)(nil),                // 5: v1beta1.Filter
	(*timestamppb.Timestamp)(nil), // 6: google.protobuf.Timestamp
}
var file_backend_api_v1beta1_filter_proto_depIdxs = []int32{
	0, // 0: v1beta1.Predicate.op:type_name -> v1beta1.Predicate.Op
	6, // 1: v1beta1.Predicate.timestamp_value:type_name -> google.protobuf.Timestamp
	2, // 2: v1beta1.Predicate.int_values:type_name -> v1beta1.IntValues
	4, // 3: v1beta1.Predicate.long_values:type_name -> v1beta1.LongValues
	3, // 4: v1beta1.Predicate.string_values:type_name -> v1beta1.StringValues
	1, // 5: v1beta1.Filter.predicates:type_name -> v1beta1.Predicate
	5, // 6: v1beta1.DummyFilterService.GetFilter:input_type -> v1beta1.Filter
	5, // 7: v1beta1.DummyFilterService.GetFilter:output_type -> v1beta1.Filter
	7, // [7:8] is the sub-list for method output_type
	6, // [6:7] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_backend_api_v1beta1_filter_proto_init() }
func file_backend_api_v1beta1_filter_proto_init() {
	if File_backend_api_v1beta1_filter_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_backend_api_v1beta1_filter_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Predicate); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_backend_api_v1beta1_filter_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IntValues); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_backend_api_v1beta1_filter_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StringValues); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_backend_api_v1beta1_filter_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LongValues); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_backend_api_v1beta1_filter_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Filter); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_backend_api_v1beta1_filter_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Predicate_IntValue)(nil),
		(*Predicate_LongValue)(nil),
		(*Predicate_StringValue)(nil),
		(*Predicate_TimestampValue)(nil),
		(*Predicate_IntValues)(nil),
		(*Predicate_LongValues)(nil),
		(*Predicate_StringValues)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_backend_api_v1beta1_filter_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_backend_api_v1beta1_filter_proto_goTypes,
		DependencyIndexes: file_backend_api_v1beta1_filter_proto_depIdxs,
		EnumInfos:         file_backend_api_v1beta1_filter_proto_enumTypes,
		MessageInfos:      file_backend_api_v1beta1_filter_proto_msgTypes,
	}.Build()
	File_backend_api_v1beta1_filter_proto = out.File
	file_backend_api_v1beta1_filter_proto_rawDesc = nil
	file_backend_api_v1beta1_filter_proto_goTypes = nil
	file_backend_api_v1beta1_filter_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// DummyFilterServiceClient is the client API for DummyFilterService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type DummyFilterServiceClient interface {
	GetFilter(ctx context.Context, in *Filter, opts ...grpc.CallOption) (*Filter, error)
}

type dummyFilterServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDummyFilterServiceClient(cc grpc.ClientConnInterface) DummyFilterServiceClient {
	return &dummyFilterServiceClient{cc}
}

func (c *dummyFilterServiceClient) GetFilter(ctx context.Context, in *Filter, opts ...grpc.CallOption) (*Filter, error) {
	out := new(Filter)
	err := c.cc.Invoke(ctx, "/v1beta1.DummyFilterService/GetFilter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DummyFilterServiceServer is the server API for DummyFilterService service.
type DummyFilterServiceServer interface {
	GetFilter(context.Context, *Filter) (*Filter, error)
}

// UnimplementedDummyFilterServiceServer can be embedded to have forward compatible implementations.
type UnimplementedDummyFilterServiceServer struct {
}

func (*UnimplementedDummyFilterServiceServer) GetFilter(context.Context, *Filter) (*Filter, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFilter not implemented")
}

func RegisterDummyFilterServiceServer(s *grpc.Server, srv DummyFilterServiceServer) {
	s.RegisterService(&_DummyFilterService_serviceDesc, srv)
}

func _DummyFilterService_GetFilter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Filter)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DummyFilterServiceServer).GetFilter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1beta1.DummyFilterService/GetFilter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DummyFilterServiceServer).GetFilter(ctx, req.(*Filter))
	}
	return interceptor(ctx, in, info, handler)
}

var _DummyFilterService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "v1beta1.DummyFilterService",
	HandlerType: (*DummyFilterServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetFilter",
			Handler:    _DummyFilterService_GetFilter_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/v1beta1/filter.proto",
}
