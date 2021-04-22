// Code generated by protoc-gen-go. DO NOT EDIT.
// source: backend/api/experiment.proto

package go_client

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Experiment_StorageState int32

const (
	Experiment_STORAGESTATE_UNSPECIFIED Experiment_StorageState = 0
	Experiment_STORAGESTATE_AVAILABLE   Experiment_StorageState = 1
	Experiment_STORAGESTATE_ARCHIVED    Experiment_StorageState = 2
)

var Experiment_StorageState_name = map[int32]string{
	0: "STORAGESTATE_UNSPECIFIED",
	1: "STORAGESTATE_AVAILABLE",
	2: "STORAGESTATE_ARCHIVED",
}

var Experiment_StorageState_value = map[string]int32{
	"STORAGESTATE_UNSPECIFIED": 0,
	"STORAGESTATE_AVAILABLE":   1,
	"STORAGESTATE_ARCHIVED":    2,
}

func (x Experiment_StorageState) String() string {
	return proto.EnumName(Experiment_StorageState_name, int32(x))
}

func (Experiment_StorageState) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{5, 0}
}

type CreateExperimentRequest struct {
	// The experiment to be created.
	Experiment           *Experiment `protobuf:"bytes,1,opt,name=experiment,proto3" json:"experiment,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *CreateExperimentRequest) Reset()         { *m = CreateExperimentRequest{} }
func (m *CreateExperimentRequest) String() string { return proto.CompactTextString(m) }
func (*CreateExperimentRequest) ProtoMessage()    {}
func (*CreateExperimentRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{0}
}

func (m *CreateExperimentRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateExperimentRequest.Unmarshal(m, b)
}
func (m *CreateExperimentRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateExperimentRequest.Marshal(b, m, deterministic)
}
func (m *CreateExperimentRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateExperimentRequest.Merge(m, src)
}
func (m *CreateExperimentRequest) XXX_Size() int {
	return xxx_messageInfo_CreateExperimentRequest.Size(m)
}
func (m *CreateExperimentRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateExperimentRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateExperimentRequest proto.InternalMessageInfo

func (m *CreateExperimentRequest) GetExperiment() *Experiment {
	if m != nil {
		return m.Experiment
	}
	return nil
}

type GetExperimentRequest struct {
	// The ID of the experiment to be retrieved.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetExperimentRequest) Reset()         { *m = GetExperimentRequest{} }
func (m *GetExperimentRequest) String() string { return proto.CompactTextString(m) }
func (*GetExperimentRequest) ProtoMessage()    {}
func (*GetExperimentRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{1}
}

func (m *GetExperimentRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetExperimentRequest.Unmarshal(m, b)
}
func (m *GetExperimentRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetExperimentRequest.Marshal(b, m, deterministic)
}
func (m *GetExperimentRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetExperimentRequest.Merge(m, src)
}
func (m *GetExperimentRequest) XXX_Size() int {
	return xxx_messageInfo_GetExperimentRequest.Size(m)
}
func (m *GetExperimentRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetExperimentRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetExperimentRequest proto.InternalMessageInfo

func (m *GetExperimentRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type ListExperimentsRequest struct {
	// A page token to request the next page of results. The token is acquried
	// from the nextPageToken field of the response from the previous
	// ListExperiment call or can be omitted when fetching the first page.
	PageToken string `protobuf:"bytes,1,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
	// The number of experiments to be listed per page. If there are more
	// experiments than this number, the response message will contain a
	// nextPageToken field you can use to fetch the next page.
	PageSize int32 `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	// Can be format of "field_name", "field_name asc" or "field_name desc"
	// Ascending by default.
	SortBy string `protobuf:"bytes,3,opt,name=sort_by,json=sortBy,proto3" json:"sort_by,omitempty"`
	// A url-encoded, JSON-serialized Filter protocol buffer (see
	// [filter.proto](https://github.com/kubeflow/pipelines/
	// blob/master/backend/api/filter.proto)).
	Filter string `protobuf:"bytes,4,opt,name=filter,proto3" json:"filter,omitempty"`
	// What resource reference to filter on.
	// For Experiment, the only valid resource type is Namespace. An sample query string could be
	// resource_reference_key.type=NAMESPACE&resource_reference_key.id=ns1
	ResourceReferenceKey *ResourceKey `protobuf:"bytes,5,opt,name=resource_reference_key,json=resourceReferenceKey,proto3" json:"resource_reference_key,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *ListExperimentsRequest) Reset()         { *m = ListExperimentsRequest{} }
func (m *ListExperimentsRequest) String() string { return proto.CompactTextString(m) }
func (*ListExperimentsRequest) ProtoMessage()    {}
func (*ListExperimentsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{2}
}

func (m *ListExperimentsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListExperimentsRequest.Unmarshal(m, b)
}
func (m *ListExperimentsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListExperimentsRequest.Marshal(b, m, deterministic)
}
func (m *ListExperimentsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListExperimentsRequest.Merge(m, src)
}
func (m *ListExperimentsRequest) XXX_Size() int {
	return xxx_messageInfo_ListExperimentsRequest.Size(m)
}
func (m *ListExperimentsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListExperimentsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListExperimentsRequest proto.InternalMessageInfo

func (m *ListExperimentsRequest) GetPageToken() string {
	if m != nil {
		return m.PageToken
	}
	return ""
}

func (m *ListExperimentsRequest) GetPageSize() int32 {
	if m != nil {
		return m.PageSize
	}
	return 0
}

func (m *ListExperimentsRequest) GetSortBy() string {
	if m != nil {
		return m.SortBy
	}
	return ""
}

func (m *ListExperimentsRequest) GetFilter() string {
	if m != nil {
		return m.Filter
	}
	return ""
}

func (m *ListExperimentsRequest) GetResourceReferenceKey() *ResourceKey {
	if m != nil {
		return m.ResourceReferenceKey
	}
	return nil
}

type ListExperimentsResponse struct {
	// A list of experiments returned.
	Experiments []*Experiment `protobuf:"bytes,1,rep,name=experiments,proto3" json:"experiments,omitempty"`
	// The total number of experiments for the given query.
	TotalSize int32 `protobuf:"varint,3,opt,name=total_size,json=totalSize,proto3" json:"total_size,omitempty"`
	// The token to list the next page of experiments.
	NextPageToken        string   `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListExperimentsResponse) Reset()         { *m = ListExperimentsResponse{} }
func (m *ListExperimentsResponse) String() string { return proto.CompactTextString(m) }
func (*ListExperimentsResponse) ProtoMessage()    {}
func (*ListExperimentsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{3}
}

func (m *ListExperimentsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListExperimentsResponse.Unmarshal(m, b)
}
func (m *ListExperimentsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListExperimentsResponse.Marshal(b, m, deterministic)
}
func (m *ListExperimentsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListExperimentsResponse.Merge(m, src)
}
func (m *ListExperimentsResponse) XXX_Size() int {
	return xxx_messageInfo_ListExperimentsResponse.Size(m)
}
func (m *ListExperimentsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListExperimentsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListExperimentsResponse proto.InternalMessageInfo

func (m *ListExperimentsResponse) GetExperiments() []*Experiment {
	if m != nil {
		return m.Experiments
	}
	return nil
}

func (m *ListExperimentsResponse) GetTotalSize() int32 {
	if m != nil {
		return m.TotalSize
	}
	return 0
}

func (m *ListExperimentsResponse) GetNextPageToken() string {
	if m != nil {
		return m.NextPageToken
	}
	return ""
}

type DeleteExperimentRequest struct {
	// The ID of the experiment to be deleted.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteExperimentRequest) Reset()         { *m = DeleteExperimentRequest{} }
func (m *DeleteExperimentRequest) String() string { return proto.CompactTextString(m) }
func (*DeleteExperimentRequest) ProtoMessage()    {}
func (*DeleteExperimentRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{4}
}

func (m *DeleteExperimentRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteExperimentRequest.Unmarshal(m, b)
}
func (m *DeleteExperimentRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteExperimentRequest.Marshal(b, m, deterministic)
}
func (m *DeleteExperimentRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteExperimentRequest.Merge(m, src)
}
func (m *DeleteExperimentRequest) XXX_Size() int {
	return xxx_messageInfo_DeleteExperimentRequest.Size(m)
}
func (m *DeleteExperimentRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteExperimentRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteExperimentRequest proto.InternalMessageInfo

func (m *DeleteExperimentRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type Experiment struct {
	// Output. Unique experiment ID. Generated by API server.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Required input field. Unique experiment name provided by user.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// Optional input field. Describing the purpose of the experiment
	Description string `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
	// Output. The time that the experiment created.
	CreatedAt *timestamp.Timestamp `protobuf:"bytes,4,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	// Optional input field. Specify which resource this run belongs to.
	// For Experiment, the only valid resource reference is a single Namespace.
	ResourceReferences []*ResourceReference `protobuf:"bytes,5,rep,name=resource_references,json=resourceReferences,proto3" json:"resource_references,omitempty"`
	// Output. Specifies whether this experiment is in archived or available state.
	StorageState         Experiment_StorageState `protobuf:"varint,6,opt,name=storage_state,json=storageState,proto3,enum=api.Experiment_StorageState" json:"storage_state,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *Experiment) Reset()         { *m = Experiment{} }
func (m *Experiment) String() string { return proto.CompactTextString(m) }
func (*Experiment) ProtoMessage()    {}
func (*Experiment) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{5}
}

func (m *Experiment) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Experiment.Unmarshal(m, b)
}
func (m *Experiment) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Experiment.Marshal(b, m, deterministic)
}
func (m *Experiment) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Experiment.Merge(m, src)
}
func (m *Experiment) XXX_Size() int {
	return xxx_messageInfo_Experiment.Size(m)
}
func (m *Experiment) XXX_DiscardUnknown() {
	xxx_messageInfo_Experiment.DiscardUnknown(m)
}

var xxx_messageInfo_Experiment proto.InternalMessageInfo

func (m *Experiment) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Experiment) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Experiment) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *Experiment) GetCreatedAt() *timestamp.Timestamp {
	if m != nil {
		return m.CreatedAt
	}
	return nil
}

func (m *Experiment) GetResourceReferences() []*ResourceReference {
	if m != nil {
		return m.ResourceReferences
	}
	return nil
}

func (m *Experiment) GetStorageState() Experiment_StorageState {
	if m != nil {
		return m.StorageState
	}
	return Experiment_STORAGESTATE_UNSPECIFIED
}

type ArchiveExperimentRequest struct {
	// The ID of the experiment to be archived.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ArchiveExperimentRequest) Reset()         { *m = ArchiveExperimentRequest{} }
func (m *ArchiveExperimentRequest) String() string { return proto.CompactTextString(m) }
func (*ArchiveExperimentRequest) ProtoMessage()    {}
func (*ArchiveExperimentRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{6}
}

func (m *ArchiveExperimentRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ArchiveExperimentRequest.Unmarshal(m, b)
}
func (m *ArchiveExperimentRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ArchiveExperimentRequest.Marshal(b, m, deterministic)
}
func (m *ArchiveExperimentRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ArchiveExperimentRequest.Merge(m, src)
}
func (m *ArchiveExperimentRequest) XXX_Size() int {
	return xxx_messageInfo_ArchiveExperimentRequest.Size(m)
}
func (m *ArchiveExperimentRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ArchiveExperimentRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ArchiveExperimentRequest proto.InternalMessageInfo

func (m *ArchiveExperimentRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type UnarchiveExperimentRequest struct {
	// The ID of the experiment to be restored.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UnarchiveExperimentRequest) Reset()         { *m = UnarchiveExperimentRequest{} }
func (m *UnarchiveExperimentRequest) String() string { return proto.CompactTextString(m) }
func (*UnarchiveExperimentRequest) ProtoMessage()    {}
func (*UnarchiveExperimentRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2acb5110e2ac785b, []int{7}
}

func (m *UnarchiveExperimentRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UnarchiveExperimentRequest.Unmarshal(m, b)
}
func (m *UnarchiveExperimentRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UnarchiveExperimentRequest.Marshal(b, m, deterministic)
}
func (m *UnarchiveExperimentRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UnarchiveExperimentRequest.Merge(m, src)
}
func (m *UnarchiveExperimentRequest) XXX_Size() int {
	return xxx_messageInfo_UnarchiveExperimentRequest.Size(m)
}
func (m *UnarchiveExperimentRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_UnarchiveExperimentRequest.DiscardUnknown(m)
}

var xxx_messageInfo_UnarchiveExperimentRequest proto.InternalMessageInfo

func (m *UnarchiveExperimentRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func init() {
	proto.RegisterEnum("api.Experiment_StorageState", Experiment_StorageState_name, Experiment_StorageState_value)
	proto.RegisterType((*CreateExperimentRequest)(nil), "api.CreateExperimentRequest")
	proto.RegisterType((*GetExperimentRequest)(nil), "api.GetExperimentRequest")
	proto.RegisterType((*ListExperimentsRequest)(nil), "api.ListExperimentsRequest")
	proto.RegisterType((*ListExperimentsResponse)(nil), "api.ListExperimentsResponse")
	proto.RegisterType((*DeleteExperimentRequest)(nil), "api.DeleteExperimentRequest")
	proto.RegisterType((*Experiment)(nil), "api.Experiment")
	proto.RegisterType((*ArchiveExperimentRequest)(nil), "api.ArchiveExperimentRequest")
	proto.RegisterType((*UnarchiveExperimentRequest)(nil), "api.UnarchiveExperimentRequest")
}

func init() { proto.RegisterFile("backend/api/experiment.proto", fileDescriptor_2acb5110e2ac785b) }

var fileDescriptor_2acb5110e2ac785b = []byte{
	// 889 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x55, 0x51, 0x53, 0xdb, 0x46,
	0x10, 0x8e, 0x4c, 0x70, 0xc2, 0x1a, 0x83, 0x39, 0x52, 0x5b, 0x08, 0x53, 0x5c, 0x4d, 0x87, 0xba,
	0x4c, 0xb0, 0x0a, 0x79, 0x6a, 0xde, 0x0c, 0x38, 0x94, 0x86, 0xb6, 0x19, 0xd9, 0xc9, 0x43, 0x5e,
	0x3c, 0x67, 0x79, 0x6d, 0x6e, 0xb0, 0x75, 0xea, 0xdd, 0x89, 0xc4, 0x74, 0x3a, 0xd3, 0xe9, 0x4c,
	0xff, 0x40, 0xf3, 0xb3, 0x3a, 0x7d, 0xea, 0x5f, 0xe8, 0xef, 0xe8, 0x74, 0x74, 0x96, 0x41, 0xb6,
	0xec, 0x84, 0x27, 0xb8, 0xdd, 0xcf, 0xb7, 0xf7, 0x7d, 0xfb, 0xed, 0x0a, 0xca, 0x1d, 0xea, 0x5d,
	0xa1, 0xdf, 0x75, 0x68, 0xc0, 0x1c, 0x7c, 0x1f, 0xa0, 0x60, 0x43, 0xf4, 0x55, 0x2d, 0x10, 0x5c,
	0x71, 0xb2, 0x44, 0x03, 0x66, 0x95, 0xa6, 0x20, 0x42, 0x70, 0x31, 0xce, 0x5a, 0x5f, 0x26, 0x13,
	0x02, 0x25, 0x0f, 0x85, 0x87, 0x6d, 0x81, 0x3d, 0x14, 0xe8, 0x7b, 0x18, 0xa3, 0xca, 0x7d, 0xce,
	0xfb, 0x03, 0xd4, 0x20, 0xea, 0xfb, 0x5c, 0x51, 0xc5, 0xb8, 0x2f, 0xe3, 0xec, 0x76, 0x9c, 0xd5,
	0xa7, 0x4e, 0xd8, 0x73, 0x70, 0x18, 0xa8, 0x51, 0x9c, 0xdc, 0x9d, 0x4d, 0x2a, 0x36, 0x44, 0xa9,
	0xe8, 0x30, 0x88, 0x01, 0x4f, 0xf5, 0x1f, 0xef, 0xa0, 0x8f, 0xfe, 0x81, 0x7c, 0x47, 0xfb, 0x7d,
	0x14, 0x0e, 0x0f, 0xf4, 0xfd, 0xe9, 0x5a, 0xf6, 0xf7, 0x50, 0x3a, 0x11, 0x48, 0x15, 0x36, 0x6e,
	0x79, 0xba, 0xf8, 0x73, 0x88, 0x52, 0x11, 0x07, 0xe0, 0x8e, 0xbc, 0x69, 0x54, 0x8c, 0x6a, 0xee,
	0x68, 0xbd, 0x46, 0x03, 0x56, 0x4b, 0x60, 0x13, 0x10, 0x7b, 0x0f, 0x9e, 0x9c, 0xa1, 0x4a, 0x5f,
	0xb4, 0x06, 0x19, 0xd6, 0xd5, 0x17, 0xac, 0xb8, 0x19, 0xd6, 0xb5, 0xff, 0x36, 0xa0, 0x78, 0xc1,
	0x64, 0x02, 0x29, 0x27, 0xd0, 0x1d, 0x80, 0x80, 0xf6, 0xb1, 0xad, 0xf8, 0x15, 0xfa, 0xf1, 0x4f,
	0x56, 0xa2, 0x48, 0x2b, 0x0a, 0x90, 0x6d, 0xd0, 0x87, 0xb6, 0x64, 0x37, 0x68, 0x66, 0x2a, 0x46,
	0x75, 0xd9, 0x7d, 0x1c, 0x05, 0x9a, 0xec, 0x06, 0x49, 0x09, 0x1e, 0x49, 0x2e, 0x54, 0xbb, 0x33,
	0x32, 0x97, 0xf4, 0x0f, 0xb3, 0xd1, 0xf1, 0x78, 0x44, 0x8a, 0x90, 0xed, 0xb1, 0x81, 0x42, 0x61,
	0x3e, 0x1c, 0xc7, 0xc7, 0x27, 0xf2, 0x02, 0x8a, 0xe9, 0x0e, 0xb5, 0xaf, 0x70, 0x64, 0x2e, 0x6b,
	0xb2, 0x05, 0x4d, 0xd6, 0x8d, 0x21, 0x2f, 0x71, 0xe4, 0x3e, 0x99, 0xe0, 0xdd, 0x09, 0xfc, 0x25,
	0x8e, 0xec, 0x0f, 0x06, 0x94, 0x52, 0x7c, 0x64, 0xc0, 0x7d, 0x89, 0xe4, 0x10, 0x72, 0x77, 0x0a,
	0x49, 0xd3, 0xa8, 0x2c, 0xcd, 0x53, 0x31, 0x89, 0x89, 0x34, 0x50, 0x5c, 0xd1, 0xc1, 0x98, 0xe5,
	0x92, 0x66, 0xb9, 0xa2, 0x23, 0x9a, 0xe6, 0x1e, 0xac, 0xfb, 0xf8, 0x5e, 0xb5, 0x13, 0x3a, 0x65,
	0x34, 0xad, 0x7c, 0x14, 0x7e, 0x35, 0xd1, 0xca, 0xfe, 0x1a, 0x4a, 0xa7, 0x38, 0xc0, 0x79, 0x9d,
	0x9d, 0x6d, 0xc8, 0x7f, 0x19, 0x80, 0x3b, 0xd4, 0x6c, 0x9a, 0x10, 0x78, 0xe8, 0xd3, 0x21, 0xc6,
	0x65, 0xf4, 0xff, 0xa4, 0x02, 0xb9, 0x2e, 0x4a, 0x4f, 0x30, 0xed, 0xac, 0x58, 0xf0, 0x64, 0x88,
	0x7c, 0x0b, 0xe0, 0x69, 0x67, 0x75, 0xdb, 0x54, 0x69, 0xe5, 0x73, 0x47, 0x56, 0x6d, 0xec, 0xde,
	0xda, 0xc4, 0xbd, 0xb5, 0xd6, 0xc4, 0xbd, 0xee, 0x4a, 0x8c, 0xae, 0x2b, 0x72, 0x06, 0x9b, 0xe9,
	0xc6, 0x48, 0x73, 0x59, 0x8b, 0x57, 0x9c, 0xea, 0xca, 0x6d, 0x23, 0x5c, 0x92, 0xea, 0x8d, 0x24,
	0x75, 0xc8, 0x4b, 0xc5, 0x85, 0xb6, 0x8c, 0xa2, 0x0a, 0xcd, 0x6c, 0xc5, 0xa8, 0xae, 0x1d, 0x95,
	0x67, 0xf4, 0xaf, 0x35, 0xc7, 0xa0, 0x66, 0x84, 0x71, 0x57, 0x65, 0xe2, 0x64, 0x7b, 0xb0, 0x9a,
	0xcc, 0x92, 0x32, 0x98, 0xcd, 0xd6, 0x4f, 0x6e, 0xfd, 0xac, 0xd1, 0x6c, 0xd5, 0x5b, 0x8d, 0xf6,
	0xeb, 0x1f, 0x9b, 0xaf, 0x1a, 0x27, 0xe7, 0x2f, 0xce, 0x1b, 0xa7, 0x85, 0x07, 0xc4, 0x82, 0xe2,
	0x54, 0xb6, 0xfe, 0xa6, 0x7e, 0x7e, 0x51, 0x3f, 0xbe, 0x68, 0x14, 0x0c, 0xb2, 0x05, 0x9f, 0x4d,
	0xe7, 0xdc, 0x93, 0xef, 0xce, 0xdf, 0x34, 0x4e, 0x0b, 0x19, 0x7b, 0x1f, 0xcc, 0xba, 0xf0, 0x2e,
	0xd9, 0xf5, 0x3d, 0x9a, 0xf5, 0x14, 0xac, 0xd7, 0x3e, 0xbd, 0x27, 0xfa, 0xe8, 0xaf, 0x65, 0xd8,
	0xb8, 0x43, 0x35, 0x51, 0x5c, 0x33, 0x0f, 0x49, 0x00, 0x85, 0xd9, 0xa9, 0x27, 0x63, 0x51, 0x16,
	0x2c, 0x03, 0x6b, 0xd6, 0xb2, 0xf6, 0xc1, 0xef, 0xff, 0xfc, 0xfb, 0x21, 0xf3, 0x95, 0xbd, 0x15,
	0x2d, 0x31, 0xe9, 0x5c, 0x1f, 0x76, 0x50, 0xd1, 0xc3, 0xc4, 0xba, 0x94, 0xcf, 0x13, 0xbb, 0x81,
	0x78, 0x90, 0x9f, 0xda, 0x0d, 0x64, 0x4b, 0x5f, 0x38, 0x6f, 0x5f, 0xa4, 0x6b, 0xed, 0xe9, 0x5a,
	0x15, 0xf2, 0xf9, 0xc2, 0x5a, 0xce, 0x2f, 0xac, 0xfb, 0x2b, 0xf1, 0x61, 0x6d, 0x7a, 0x0e, 0xc9,
	0xb6, 0xbe, 0x6a, 0xfe, 0xb2, 0xb1, 0xca, 0xf3, 0x93, 0xe3, 0xc9, 0xb5, 0xbf, 0xd0, 0x45, 0xb7,
	0xc9, 0x62, 0x82, 0x91, 0x8c, 0xb3, 0x23, 0x16, 0xcb, 0xb8, 0x60, 0xf2, 0xac, 0x62, 0x6a, 0x00,
	0x1a, 0xd1, 0x6e, 0x9f, 0x30, 0xdc, 0xff, 0x14, 0xc3, 0x1b, 0xd8, 0x48, 0x19, 0x85, 0xec, 0xe8,
	0x92, 0x8b, 0x0c, 0xb4, 0xb0, 0x66, 0x4d, 0xd7, 0xac, 0xda, 0x7b, 0x1f, 0xaf, 0xf9, 0x3c, 0xf6,
	0x1a, 0xf9, 0xcd, 0x80, 0xcd, 0x39, 0xce, 0x23, 0xbb, 0xba, 0xfc, 0x62, 0x4f, 0x2e, 0x7c, 0xc0,
	0x37, 0xfa, 0x01, 0xfb, 0x76, 0xf5, 0x13, 0x0f, 0x08, 0x27, 0x57, 0x1f, 0xff, 0x61, 0xfc, 0x59,
	0xff, 0xc1, 0x2d, 0xc3, 0xa3, 0x2e, 0xf6, 0x68, 0x38, 0x50, 0x64, 0x83, 0xac, 0x43, 0xde, 0xca,
	0xe9, 0x17, 0x44, 0xf3, 0x19, 0xca, 0xb7, 0xbb, 0xb0, 0x03, 0xd9, 0x63, 0xa4, 0x02, 0x05, 0xd9,
	0x7c, 0x9c, 0xb1, 0xf2, 0x34, 0x54, 0x97, 0x5c, 0xb0, 0x1b, 0xfd, 0xdd, 0xab, 0x64, 0x3a, 0xab,
	0x00, 0xb7, 0x80, 0x07, 0x6f, 0x9f, 0xf5, 0x99, 0xba, 0x0c, 0x3b, 0x35, 0x8f, 0x0f, 0x9d, 0xab,
	0xb0, 0x83, 0xbd, 0x01, 0x7f, 0xe7, 0x04, 0x2c, 0xc0, 0x01, 0xf3, 0x51, 0x3a, 0xc9, 0xcf, 0x79,
	0x9f, 0xb7, 0xbd, 0x01, 0x43, 0x5f, 0x75, 0xb2, 0x9a, 0xc9, 0xb3, 0xff, 0x03, 0x00, 0x00, 0xff,
	0xff, 0x0b, 0x8f, 0xfa, 0xad, 0x2a, 0x08, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ExperimentServiceClient is the client API for ExperimentService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ExperimentServiceClient interface {
	// Creates a new experiment.
	CreateExperiment(ctx context.Context, in *CreateExperimentRequest, opts ...grpc.CallOption) (*Experiment, error)
	// Finds a specific experiment by ID.
	GetExperiment(ctx context.Context, in *GetExperimentRequest, opts ...grpc.CallOption) (*Experiment, error)
	// Finds all experiments. Supports pagination, and sorting on certain fields.
	ListExperiment(ctx context.Context, in *ListExperimentsRequest, opts ...grpc.CallOption) (*ListExperimentsResponse, error)
	// Deletes an experiment without deleting the experiment's runs and jobs. To
	// avoid unexpected behaviors, delete an experiment's runs and jobs before
	// deleting the experiment.
	DeleteExperiment(ctx context.Context, in *DeleteExperimentRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Archives an experiment and the experiment's runs and jobs.
	ArchiveExperiment(ctx context.Context, in *ArchiveExperimentRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Restores an archived experiment. The experiment's archived runs and jobs
	// will stay archived.
	UnarchiveExperiment(ctx context.Context, in *UnarchiveExperimentRequest, opts ...grpc.CallOption) (*empty.Empty, error)
}

type experimentServiceClient struct {
	cc *grpc.ClientConn
}

func NewExperimentServiceClient(cc *grpc.ClientConn) ExperimentServiceClient {
	return &experimentServiceClient{cc}
}

func (c *experimentServiceClient) CreateExperiment(ctx context.Context, in *CreateExperimentRequest, opts ...grpc.CallOption) (*Experiment, error) {
	out := new(Experiment)
	err := c.cc.Invoke(ctx, "/api.ExperimentService/CreateExperiment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *experimentServiceClient) GetExperiment(ctx context.Context, in *GetExperimentRequest, opts ...grpc.CallOption) (*Experiment, error) {
	out := new(Experiment)
	err := c.cc.Invoke(ctx, "/api.ExperimentService/GetExperiment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *experimentServiceClient) ListExperiment(ctx context.Context, in *ListExperimentsRequest, opts ...grpc.CallOption) (*ListExperimentsResponse, error) {
	out := new(ListExperimentsResponse)
	err := c.cc.Invoke(ctx, "/api.ExperimentService/ListExperiment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *experimentServiceClient) DeleteExperiment(ctx context.Context, in *DeleteExperimentRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.ExperimentService/DeleteExperiment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *experimentServiceClient) ArchiveExperiment(ctx context.Context, in *ArchiveExperimentRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.ExperimentService/ArchiveExperiment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *experimentServiceClient) UnarchiveExperiment(ctx context.Context, in *UnarchiveExperimentRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.ExperimentService/UnarchiveExperiment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ExperimentServiceServer is the server API for ExperimentService service.
type ExperimentServiceServer interface {
	// Creates a new experiment.
	CreateExperiment(context.Context, *CreateExperimentRequest) (*Experiment, error)
	// Finds a specific experiment by ID.
	GetExperiment(context.Context, *GetExperimentRequest) (*Experiment, error)
	// Finds all experiments. Supports pagination, and sorting on certain fields.
	ListExperiment(context.Context, *ListExperimentsRequest) (*ListExperimentsResponse, error)
	// Deletes an experiment without deleting the experiment's runs and jobs. To
	// avoid unexpected behaviors, delete an experiment's runs and jobs before
	// deleting the experiment.
	DeleteExperiment(context.Context, *DeleteExperimentRequest) (*empty.Empty, error)
	// Archives an experiment and the experiment's runs and jobs.
	ArchiveExperiment(context.Context, *ArchiveExperimentRequest) (*empty.Empty, error)
	// Restores an archived experiment. The experiment's archived runs and jobs
	// will stay archived.
	UnarchiveExperiment(context.Context, *UnarchiveExperimentRequest) (*empty.Empty, error)
}

// UnimplementedExperimentServiceServer can be embedded to have forward compatible implementations.
type UnimplementedExperimentServiceServer struct {
}

func (*UnimplementedExperimentServiceServer) CreateExperiment(ctx context.Context, req *CreateExperimentRequest) (*Experiment, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateExperiment not implemented")
}
func (*UnimplementedExperimentServiceServer) GetExperiment(ctx context.Context, req *GetExperimentRequest) (*Experiment, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetExperiment not implemented")
}
func (*UnimplementedExperimentServiceServer) ListExperiment(ctx context.Context, req *ListExperimentsRequest) (*ListExperimentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListExperiment not implemented")
}
func (*UnimplementedExperimentServiceServer) DeleteExperiment(ctx context.Context, req *DeleteExperimentRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteExperiment not implemented")
}
func (*UnimplementedExperimentServiceServer) ArchiveExperiment(ctx context.Context, req *ArchiveExperimentRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ArchiveExperiment not implemented")
}
func (*UnimplementedExperimentServiceServer) UnarchiveExperiment(ctx context.Context, req *UnarchiveExperimentRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnarchiveExperiment not implemented")
}

func RegisterExperimentServiceServer(s *grpc.Server, srv ExperimentServiceServer) {
	s.RegisterService(&_ExperimentService_serviceDesc, srv)
}

func _ExperimentService_CreateExperiment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateExperimentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExperimentServiceServer).CreateExperiment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ExperimentService/CreateExperiment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExperimentServiceServer).CreateExperiment(ctx, req.(*CreateExperimentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ExperimentService_GetExperiment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetExperimentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExperimentServiceServer).GetExperiment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ExperimentService/GetExperiment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExperimentServiceServer).GetExperiment(ctx, req.(*GetExperimentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ExperimentService_ListExperiment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListExperimentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExperimentServiceServer).ListExperiment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ExperimentService/ListExperiment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExperimentServiceServer).ListExperiment(ctx, req.(*ListExperimentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ExperimentService_DeleteExperiment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteExperimentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExperimentServiceServer).DeleteExperiment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ExperimentService/DeleteExperiment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExperimentServiceServer).DeleteExperiment(ctx, req.(*DeleteExperimentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ExperimentService_ArchiveExperiment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ArchiveExperimentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExperimentServiceServer).ArchiveExperiment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ExperimentService/ArchiveExperiment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExperimentServiceServer).ArchiveExperiment(ctx, req.(*ArchiveExperimentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ExperimentService_UnarchiveExperiment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnarchiveExperimentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExperimentServiceServer).UnarchiveExperiment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ExperimentService/UnarchiveExperiment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExperimentServiceServer).UnarchiveExperiment(ctx, req.(*UnarchiveExperimentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ExperimentService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.ExperimentService",
	HandlerType: (*ExperimentServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateExperiment",
			Handler:    _ExperimentService_CreateExperiment_Handler,
		},
		{
			MethodName: "GetExperiment",
			Handler:    _ExperimentService_GetExperiment_Handler,
		},
		{
			MethodName: "ListExperiment",
			Handler:    _ExperimentService_ListExperiment_Handler,
		},
		{
			MethodName: "DeleteExperiment",
			Handler:    _ExperimentService_DeleteExperiment_Handler,
		},
		{
			MethodName: "ArchiveExperiment",
			Handler:    _ExperimentService_ArchiveExperiment_Handler,
		},
		{
			MethodName: "UnarchiveExperiment",
			Handler:    _ExperimentService_UnarchiveExperiment_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/experiment.proto",
}
