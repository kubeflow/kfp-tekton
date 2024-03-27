// Copyright 2020 The Kubeflow Authors
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
// 	protoc-gen-go v1.33.0
// 	protoc        v3.17.3
// source: backend/api/v1beta1/auth.proto

package go_client

import (
	context "context"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Type of resources in pipelines system.
type AuthorizeRequest_Resources int32

const (
	AuthorizeRequest_UNASSIGNED_RESOURCES AuthorizeRequest_Resources = 0
	AuthorizeRequest_VIEWERS              AuthorizeRequest_Resources = 1
)

// Enum value maps for AuthorizeRequest_Resources.
var (
	AuthorizeRequest_Resources_name = map[int32]string{
		0: "UNASSIGNED_RESOURCES",
		1: "VIEWERS",
	}
	AuthorizeRequest_Resources_value = map[string]int32{
		"UNASSIGNED_RESOURCES": 0,
		"VIEWERS":              1,
	}
)

func (x AuthorizeRequest_Resources) Enum() *AuthorizeRequest_Resources {
	p := new(AuthorizeRequest_Resources)
	*p = x
	return p
}

func (x AuthorizeRequest_Resources) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AuthorizeRequest_Resources) Descriptor() protoreflect.EnumDescriptor {
	return file_backend_api_v1beta1_auth_proto_enumTypes[0].Descriptor()
}

func (AuthorizeRequest_Resources) Type() protoreflect.EnumType {
	return &file_backend_api_v1beta1_auth_proto_enumTypes[0]
}

func (x AuthorizeRequest_Resources) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AuthorizeRequest_Resources.Descriptor instead.
func (AuthorizeRequest_Resources) EnumDescriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_auth_proto_rawDescGZIP(), []int{0, 0}
}

// Type of verbs that act on the resources.
type AuthorizeRequest_Verb int32

const (
	AuthorizeRequest_UNASSIGNED_VERB AuthorizeRequest_Verb = 0
	AuthorizeRequest_CREATE          AuthorizeRequest_Verb = 1
	AuthorizeRequest_GET             AuthorizeRequest_Verb = 2
	AuthorizeRequest_DELETE          AuthorizeRequest_Verb = 3
)

// Enum value maps for AuthorizeRequest_Verb.
var (
	AuthorizeRequest_Verb_name = map[int32]string{
		0: "UNASSIGNED_VERB",
		1: "CREATE",
		2: "GET",
		3: "DELETE",
	}
	AuthorizeRequest_Verb_value = map[string]int32{
		"UNASSIGNED_VERB": 0,
		"CREATE":          1,
		"GET":             2,
		"DELETE":          3,
	}
)

func (x AuthorizeRequest_Verb) Enum() *AuthorizeRequest_Verb {
	p := new(AuthorizeRequest_Verb)
	*p = x
	return p
}

func (x AuthorizeRequest_Verb) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AuthorizeRequest_Verb) Descriptor() protoreflect.EnumDescriptor {
	return file_backend_api_v1beta1_auth_proto_enumTypes[1].Descriptor()
}

func (AuthorizeRequest_Verb) Type() protoreflect.EnumType {
	return &file_backend_api_v1beta1_auth_proto_enumTypes[1]
}

func (x AuthorizeRequest_Verb) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AuthorizeRequest_Verb.Descriptor instead.
func (AuthorizeRequest_Verb) EnumDescriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_auth_proto_rawDescGZIP(), []int{0, 1}
}

// Ask for authorization of an access by providing resource's namespace, type
// and verb. User identity is not part of the message, because it is expected
// to be parsed from request headers. Caller should proxy user request's headers.
type AuthorizeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string                     `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`                                      // Namespace the resource belongs to.
	Resources AuthorizeRequest_Resources `protobuf:"varint,2,opt,name=resources,proto3,enum=api.AuthorizeRequest_Resources" json:"resources,omitempty"` // Resource type asking for authorization.
	Verb      AuthorizeRequest_Verb      `protobuf:"varint,3,opt,name=verb,proto3,enum=api.AuthorizeRequest_Verb" json:"verb,omitempty"`                // Verb on the resource asking for authorization.
}

func (x *AuthorizeRequest) Reset() {
	*x = AuthorizeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v1beta1_auth_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AuthorizeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthorizeRequest) ProtoMessage() {}

func (x *AuthorizeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v1beta1_auth_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthorizeRequest.ProtoReflect.Descriptor instead.
func (*AuthorizeRequest) Descriptor() ([]byte, []int) {
	return file_backend_api_v1beta1_auth_proto_rawDescGZIP(), []int{0}
}

func (x *AuthorizeRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *AuthorizeRequest) GetResources() AuthorizeRequest_Resources {
	if x != nil {
		return x.Resources
	}
	return AuthorizeRequest_UNASSIGNED_RESOURCES
}

func (x *AuthorizeRequest) GetVerb() AuthorizeRequest_Verb {
	if x != nil {
		return x.Verb
	}
	return AuthorizeRequest_UNASSIGNED_VERB
}

var File_backend_api_v1beta1_auth_proto protoreflect.FileDescriptor

var file_backend_api_v1beta1_auth_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31,
	0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x03, 0x61, 0x70, 0x69, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x2c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x73, 0x77, 0x61,
	0x67, 0x67, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e,
	0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x91,
	0x02, 0x0a, 0x10, 0x41, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x12, 0x3d, 0x0a, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x1f, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x6f,
	0x72, 0x69, 0x7a, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x52, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x73, 0x52, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73,
	0x12, 0x2e, 0x0a, 0x04, 0x76, 0x65, 0x72, 0x62, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1a,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x56, 0x65, 0x72, 0x62, 0x52, 0x04, 0x76, 0x65, 0x72, 0x62,
	0x22, 0x32, 0x0a, 0x09, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x12, 0x18, 0x0a,
	0x14, 0x55, 0x4e, 0x41, 0x53, 0x53, 0x49, 0x47, 0x4e, 0x45, 0x44, 0x5f, 0x52, 0x45, 0x53, 0x4f,
	0x55, 0x52, 0x43, 0x45, 0x53, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x56, 0x49, 0x45, 0x57, 0x45,
	0x52, 0x53, 0x10, 0x01, 0x22, 0x3c, 0x0a, 0x04, 0x56, 0x65, 0x72, 0x62, 0x12, 0x13, 0x0a, 0x0f,
	0x55, 0x4e, 0x41, 0x53, 0x53, 0x49, 0x47, 0x4e, 0x45, 0x44, 0x5f, 0x56, 0x45, 0x52, 0x42, 0x10,
	0x00, 0x12, 0x0a, 0x0a, 0x06, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x10, 0x01, 0x12, 0x07, 0x0a,
	0x03, 0x47, 0x45, 0x54, 0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x45, 0x4c, 0x45, 0x54, 0x45,
	0x10, 0x03, 0x32, 0x67, 0x0a, 0x0b, 0x41, 0x75, 0x74, 0x68, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x12, 0x58, 0x0a, 0x0b, 0x41, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x65, 0x56, 0x31,
	0x12, 0x15, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x65,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22,
	0x1a, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x14, 0x12, 0x12, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76,
	0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x42, 0x8d, 0x01, 0x92, 0x41,
	0x4d, 0x52, 0x1c, 0x0a, 0x07, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x12, 0x11, 0x12, 0x0f,
	0x0a, 0x0d, 0x1a, 0x0b, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5a,
	0x1f, 0x0a, 0x1d, 0x0a, 0x06, 0x42, 0x65, 0x61, 0x72, 0x65, 0x72, 0x12, 0x13, 0x08, 0x02, 0x1a,
	0x0d, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x02,
	0x62, 0x0c, 0x0a, 0x0a, 0x0a, 0x06, 0x42, 0x65, 0x61, 0x72, 0x65, 0x72, 0x12, 0x00, 0x5a, 0x3b,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x66,
	0x6c, 0x6f, 0x77, 0x2f, 0x70, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2f, 0x62, 0x61,
	0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61,
	0x31, 0x2f, 0x67, 0x6f, 0x5f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_backend_api_v1beta1_auth_proto_rawDescOnce sync.Once
	file_backend_api_v1beta1_auth_proto_rawDescData = file_backend_api_v1beta1_auth_proto_rawDesc
)

func file_backend_api_v1beta1_auth_proto_rawDescGZIP() []byte {
	file_backend_api_v1beta1_auth_proto_rawDescOnce.Do(func() {
		file_backend_api_v1beta1_auth_proto_rawDescData = protoimpl.X.CompressGZIP(file_backend_api_v1beta1_auth_proto_rawDescData)
	})
	return file_backend_api_v1beta1_auth_proto_rawDescData
}

var file_backend_api_v1beta1_auth_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_backend_api_v1beta1_auth_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_backend_api_v1beta1_auth_proto_goTypes = []interface{}{
	(AuthorizeRequest_Resources)(0), // 0: api.AuthorizeRequest.Resources
	(AuthorizeRequest_Verb)(0),      // 1: api.AuthorizeRequest.Verb
	(*AuthorizeRequest)(nil),        // 2: api.AuthorizeRequest
	(*emptypb.Empty)(nil),           // 3: google.protobuf.Empty
}
var file_backend_api_v1beta1_auth_proto_depIdxs = []int32{
	0, // 0: api.AuthorizeRequest.resources:type_name -> api.AuthorizeRequest.Resources
	1, // 1: api.AuthorizeRequest.verb:type_name -> api.AuthorizeRequest.Verb
	2, // 2: api.AuthService.AuthorizeV1:input_type -> api.AuthorizeRequest
	3, // 3: api.AuthService.AuthorizeV1:output_type -> google.protobuf.Empty
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_backend_api_v1beta1_auth_proto_init() }
func file_backend_api_v1beta1_auth_proto_init() {
	if File_backend_api_v1beta1_auth_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_backend_api_v1beta1_auth_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AuthorizeRequest); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_backend_api_v1beta1_auth_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_backend_api_v1beta1_auth_proto_goTypes,
		DependencyIndexes: file_backend_api_v1beta1_auth_proto_depIdxs,
		EnumInfos:         file_backend_api_v1beta1_auth_proto_enumTypes,
		MessageInfos:      file_backend_api_v1beta1_auth_proto_msgTypes,
	}.Build()
	File_backend_api_v1beta1_auth_proto = out.File
	file_backend_api_v1beta1_auth_proto_rawDesc = nil
	file_backend_api_v1beta1_auth_proto_goTypes = nil
	file_backend_api_v1beta1_auth_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// AuthServiceClient is the client API for AuthService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AuthServiceClient interface {
	AuthorizeV1(ctx context.Context, in *AuthorizeRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type authServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthServiceClient(cc grpc.ClientConnInterface) AuthServiceClient {
	return &authServiceClient{cc}
}

func (c *authServiceClient) AuthorizeV1(ctx context.Context, in *AuthorizeRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/api.AuthService/AuthorizeV1", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuthServiceServer is the server API for AuthService service.
type AuthServiceServer interface {
	AuthorizeV1(context.Context, *AuthorizeRequest) (*emptypb.Empty, error)
}

// UnimplementedAuthServiceServer can be embedded to have forward compatible implementations.
type UnimplementedAuthServiceServer struct {
}

func (*UnimplementedAuthServiceServer) AuthorizeV1(context.Context, *AuthorizeRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AuthorizeV1 not implemented")
}

func RegisterAuthServiceServer(s *grpc.Server, srv AuthServiceServer) {
	s.RegisterService(&_AuthService_serviceDesc, srv)
}

func _AuthService_AuthorizeV1_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthorizeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServiceServer).AuthorizeV1(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AuthService/AuthorizeV1",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServiceServer).AuthorizeV1(ctx, req.(*AuthorizeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _AuthService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.AuthService",
	HandlerType: (*AuthServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AuthorizeV1",
			Handler:    _AuthService_AuthorizeV1_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/v1beta1/auth.proto",
}
