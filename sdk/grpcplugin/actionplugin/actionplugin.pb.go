// Code generated by protoc-gen-go. DO NOT EDIT.
// source: actionplugin.proto

package actionplugin

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
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
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ActionPluginManifest struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Version              string   `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	Description          string   `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
	Author               string   `protobuf:"bytes,4,opt,name=author,proto3" json:"author,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ActionPluginManifest) Reset()         { *m = ActionPluginManifest{} }
func (m *ActionPluginManifest) String() string { return proto.CompactTextString(m) }
func (*ActionPluginManifest) ProtoMessage()    {}
func (*ActionPluginManifest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8761e3c72e0ffc53, []int{0}
}

func (m *ActionPluginManifest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ActionPluginManifest.Unmarshal(m, b)
}
func (m *ActionPluginManifest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ActionPluginManifest.Marshal(b, m, deterministic)
}
func (m *ActionPluginManifest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ActionPluginManifest.Merge(m, src)
}
func (m *ActionPluginManifest) XXX_Size() int {
	return xxx_messageInfo_ActionPluginManifest.Size(m)
}
func (m *ActionPluginManifest) XXX_DiscardUnknown() {
	xxx_messageInfo_ActionPluginManifest.DiscardUnknown(m)
}

var xxx_messageInfo_ActionPluginManifest proto.InternalMessageInfo

func (m *ActionPluginManifest) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *ActionPluginManifest) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

func (m *ActionPluginManifest) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *ActionPluginManifest) GetAuthor() string {
	if m != nil {
		return m.Author
	}
	return ""
}

type ActionQuery struct {
	Options              map[string]string `protobuf:"bytes,1,rep,name=options,proto3" json:"options,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	JobID                int64             `protobuf:"varint,2,opt,name=jobID,proto3" json:"jobID,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ActionQuery) Reset()         { *m = ActionQuery{} }
func (m *ActionQuery) String() string { return proto.CompactTextString(m) }
func (*ActionQuery) ProtoMessage()    {}
func (*ActionQuery) Descriptor() ([]byte, []int) {
	return fileDescriptor_8761e3c72e0ffc53, []int{1}
}

func (m *ActionQuery) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ActionQuery.Unmarshal(m, b)
}
func (m *ActionQuery) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ActionQuery.Marshal(b, m, deterministic)
}
func (m *ActionQuery) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ActionQuery.Merge(m, src)
}
func (m *ActionQuery) XXX_Size() int {
	return xxx_messageInfo_ActionQuery.Size(m)
}
func (m *ActionQuery) XXX_DiscardUnknown() {
	xxx_messageInfo_ActionQuery.DiscardUnknown(m)
}

var xxx_messageInfo_ActionQuery proto.InternalMessageInfo

func (m *ActionQuery) GetOptions() map[string]string {
	if m != nil {
		return m.Options
	}
	return nil
}

func (m *ActionQuery) GetJobID() int64 {
	if m != nil {
		return m.JobID
	}
	return 0
}

type ActionResult struct {
	Status               string   `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	Details              string   `protobuf:"bytes,2,opt,name=details,proto3" json:"details,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ActionResult) Reset()         { *m = ActionResult{} }
func (m *ActionResult) String() string { return proto.CompactTextString(m) }
func (*ActionResult) ProtoMessage()    {}
func (*ActionResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_8761e3c72e0ffc53, []int{2}
}

func (m *ActionResult) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ActionResult.Unmarshal(m, b)
}
func (m *ActionResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ActionResult.Marshal(b, m, deterministic)
}
func (m *ActionResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ActionResult.Merge(m, src)
}
func (m *ActionResult) XXX_Size() int {
	return xxx_messageInfo_ActionResult.Size(m)
}
func (m *ActionResult) XXX_DiscardUnknown() {
	xxx_messageInfo_ActionResult.DiscardUnknown(m)
}

var xxx_messageInfo_ActionResult proto.InternalMessageInfo

func (m *ActionResult) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

func (m *ActionResult) GetDetails() string {
	if m != nil {
		return m.Details
	}
	return ""
}

type WorkerHTTPPortQuery struct {
	Port                 int32    `protobuf:"varint,1,opt,name=port,proto3" json:"port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *WorkerHTTPPortQuery) Reset()         { *m = WorkerHTTPPortQuery{} }
func (m *WorkerHTTPPortQuery) String() string { return proto.CompactTextString(m) }
func (*WorkerHTTPPortQuery) ProtoMessage()    {}
func (*WorkerHTTPPortQuery) Descriptor() ([]byte, []int) {
	return fileDescriptor_8761e3c72e0ffc53, []int{3}
}

func (m *WorkerHTTPPortQuery) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_WorkerHTTPPortQuery.Unmarshal(m, b)
}
func (m *WorkerHTTPPortQuery) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_WorkerHTTPPortQuery.Marshal(b, m, deterministic)
}
func (m *WorkerHTTPPortQuery) XXX_Merge(src proto.Message) {
	xxx_messageInfo_WorkerHTTPPortQuery.Merge(m, src)
}
func (m *WorkerHTTPPortQuery) XXX_Size() int {
	return xxx_messageInfo_WorkerHTTPPortQuery.Size(m)
}
func (m *WorkerHTTPPortQuery) XXX_DiscardUnknown() {
	xxx_messageInfo_WorkerHTTPPortQuery.DiscardUnknown(m)
}

var xxx_messageInfo_WorkerHTTPPortQuery proto.InternalMessageInfo

func (m *WorkerHTTPPortQuery) GetPort() int32 {
	if m != nil {
		return m.Port
	}
	return 0
}

func init() {
	proto.RegisterType((*ActionPluginManifest)(nil), "actionplugin.ActionPluginManifest")
	proto.RegisterType((*ActionQuery)(nil), "actionplugin.ActionQuery")
	proto.RegisterMapType((map[string]string)(nil), "actionplugin.ActionQuery.OptionsEntry")
	proto.RegisterType((*ActionResult)(nil), "actionplugin.ActionResult")
	proto.RegisterType((*WorkerHTTPPortQuery)(nil), "actionplugin.WorkerHTTPPortQuery")
}

func init() { proto.RegisterFile("actionplugin.proto", fileDescriptor_8761e3c72e0ffc53) }

var fileDescriptor_8761e3c72e0ffc53 = []byte{
	// 425 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x53, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0x8d, 0x63, 0xb7, 0x85, 0x49, 0x84, 0x60, 0xa8, 0x2a, 0x63, 0x2e, 0x61, 0x0f, 0x50, 0x2e,
	0x5b, 0xa9, 0x5c, 0xaa, 0x1e, 0x50, 0xa9, 0xa8, 0x54, 0x24, 0x2a, 0x8c, 0xa9, 0x84, 0xc4, 0xcd,
	0xb1, 0x37, 0x8e, 0x89, 0xe3, 0xb5, 0xf6, 0x23, 0x92, 0x2f, 0xfc, 0x97, 0xfc, 0x53, 0xe4, 0x5d,
	0xbb, 0xb2, 0xa5, 0xf8, 0x36, 0x6f, 0xe6, 0xcd, 0xec, 0xbc, 0x79, 0x5a, 0xc0, 0x38, 0x51, 0x39,
	0x2f, 0xab, 0x42, 0x67, 0x79, 0x49, 0x2b, 0xc1, 0x15, 0xc7, 0x79, 0x3f, 0x17, 0xbc, 0xcd, 0x38,
	0xcf, 0x0a, 0x76, 0x61, 0x6a, 0x4b, 0xbd, 0xba, 0x60, 0xdb, 0x4a, 0xd5, 0x96, 0x4a, 0xfe, 0xc1,
	0xe9, 0x17, 0x43, 0x0e, 0x0d, 0xf9, 0x21, 0x2e, 0xf3, 0x15, 0x93, 0x0a, 0x11, 0xbc, 0x32, 0xde,
	0x32, 0xdf, 0x59, 0x38, 0xe7, 0xcf, 0x23, 0x13, 0xa3, 0x0f, 0x27, 0x3b, 0x26, 0x64, 0xce, 0x4b,
	0x7f, 0x6a, 0xd2, 0x1d, 0xc4, 0x05, 0xcc, 0x52, 0x26, 0x13, 0x91, 0x57, 0xcd, 0x28, 0xdf, 0x35,
	0xd5, 0x7e, 0x0a, 0xcf, 0xe0, 0x38, 0xd6, 0x6a, 0xcd, 0x85, 0xef, 0x99, 0x62, 0x8b, 0xc8, 0xde,
	0x81, 0x99, 0x5d, 0xe0, 0xa7, 0x66, 0xa2, 0xc6, 0x1b, 0x38, 0xe1, 0xa6, 0x43, 0xfa, 0xce, 0xc2,
	0x3d, 0x9f, 0x5d, 0xbe, 0xa7, 0x03, 0x81, 0x3d, 0x2e, 0xfd, 0x61, 0x89, 0x77, 0xa5, 0x12, 0x75,
	0xd4, 0xb5, 0xe1, 0x29, 0x1c, 0xfd, 0xe5, 0xcb, 0x6f, 0x5f, 0xcd, 0x8e, 0x6e, 0x64, 0x41, 0x70,
	0x0d, 0xf3, 0x3e, 0x1d, 0x5f, 0x82, 0xbb, 0x61, 0x75, 0x2b, 0xaf, 0x09, 0x9b, 0xbe, 0x5d, 0x5c,
	0x68, 0xd6, 0x6a, 0xb3, 0xe0, 0x7a, 0x7a, 0xe5, 0x90, 0x1b, 0x98, 0xdb, 0x67, 0x23, 0x26, 0x75,
	0xa1, 0x1a, 0x2d, 0x52, 0xc5, 0x4a, 0xcb, 0xb6, 0xbd, 0x45, 0xcd, 0x7d, 0x52, 0xa6, 0xe2, 0xbc,
	0x90, 0xdd, 0x7d, 0x5a, 0x48, 0x3e, 0xc2, 0xeb, 0xdf, 0x5c, 0x6c, 0x98, 0xb8, 0x7f, 0x7c, 0x0c,
	0x43, 0x2e, 0x94, 0x15, 0x8b, 0xe0, 0x55, 0x5c, 0x28, 0x33, 0xe6, 0x28, 0x32, 0xf1, 0xe5, 0x7e,
	0xda, 0xbd, 0x66, 0x1d, 0xc1, 0x7b, 0x78, 0xf6, 0xe4, 0xca, 0x19, 0xb5, 0x5e, 0xd2, 0xce, 0x4b,
	0x7a, 0xd7, 0x78, 0x19, 0x90, 0x43, 0x47, 0x1a, 0x3a, 0x4a, 0x26, 0xf8, 0x19, 0xdc, 0x48, 0x97,
	0xf8, 0x66, 0xf4, 0xa2, 0x41, 0x70, 0xa8, 0x64, 0x55, 0x93, 0x09, 0x3e, 0xc0, 0x8b, 0xa1, 0x0a,
	0x7c, 0x37, 0xe4, 0x1f, 0xd0, 0x18, 0x8c, 0xac, 0x4c, 0x26, 0x78, 0x05, 0xde, 0x2f, 0xc5, 0xab,
	0x51, 0x51, 0xa3, 0x9d, 0xb7, 0xdf, 0xe1, 0x43, 0xc2, 0xb7, 0x94, 0xef, 0xd6, 0x34, 0x49, 0x25,
	0x95, 0xe9, 0x86, 0x66, 0xa2, 0x4a, 0xda, 0x2d, 0xfa, 0x2b, 0xdd, 0xbe, 0xea, 0xdf, 0x22, 0x6c,
	0x06, 0x85, 0xce, 0x9f, 0xc1, 0xff, 0x58, 0x1e, 0x9b, 0xf9, 0x9f, 0xfe, 0x07, 0x00, 0x00, 0xff,
	0xff, 0x00, 0x7f, 0x06, 0xee, 0x4a, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ActionPluginClient is the client API for ActionPlugin service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ActionPluginClient interface {
	Manifest(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*ActionPluginManifest, error)
	Run(ctx context.Context, in *ActionQuery, opts ...grpc.CallOption) (*ActionResult, error)
	WorkerHTTPPort(ctx context.Context, in *WorkerHTTPPortQuery, opts ...grpc.CallOption) (*empty.Empty, error)
	Stop(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*empty.Empty, error)
}

type actionPluginClient struct {
	cc *grpc.ClientConn
}

func NewActionPluginClient(cc *grpc.ClientConn) ActionPluginClient {
	return &actionPluginClient{cc}
}

func (c *actionPluginClient) Manifest(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*ActionPluginManifest, error) {
	out := new(ActionPluginManifest)
	err := c.cc.Invoke(ctx, "/actionplugin.ActionPlugin/Manifest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *actionPluginClient) Run(ctx context.Context, in *ActionQuery, opts ...grpc.CallOption) (*ActionResult, error) {
	out := new(ActionResult)
	err := c.cc.Invoke(ctx, "/actionplugin.ActionPlugin/Run", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *actionPluginClient) WorkerHTTPPort(ctx context.Context, in *WorkerHTTPPortQuery, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/actionplugin.ActionPlugin/WorkerHTTPPort", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *actionPluginClient) Stop(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/actionplugin.ActionPlugin/Stop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ActionPluginServer is the server API for ActionPlugin service.
type ActionPluginServer interface {
	Manifest(context.Context, *empty.Empty) (*ActionPluginManifest, error)
	Run(context.Context, *ActionQuery) (*ActionResult, error)
	WorkerHTTPPort(context.Context, *WorkerHTTPPortQuery) (*empty.Empty, error)
	Stop(context.Context, *empty.Empty) (*empty.Empty, error)
}

func RegisterActionPluginServer(s *grpc.Server, srv ActionPluginServer) {
	s.RegisterService(&_ActionPlugin_serviceDesc, srv)
}

func _ActionPlugin_Manifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ActionPluginServer).Manifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/actionplugin.ActionPlugin/Manifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ActionPluginServer).Manifest(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ActionPlugin_Run_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ActionQuery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ActionPluginServer).Run(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/actionplugin.ActionPlugin/Run",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ActionPluginServer).Run(ctx, req.(*ActionQuery))
	}
	return interceptor(ctx, in, info, handler)
}

func _ActionPlugin_WorkerHTTPPort_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WorkerHTTPPortQuery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ActionPluginServer).WorkerHTTPPort(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/actionplugin.ActionPlugin/WorkerHTTPPort",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ActionPluginServer).WorkerHTTPPort(ctx, req.(*WorkerHTTPPortQuery))
	}
	return interceptor(ctx, in, info, handler)
}

func _ActionPlugin_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ActionPluginServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/actionplugin.ActionPlugin/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ActionPluginServer).Stop(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _ActionPlugin_serviceDesc = grpc.ServiceDesc{
	ServiceName: "actionplugin.ActionPlugin",
	HandlerType: (*ActionPluginServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Manifest",
			Handler:    _ActionPlugin_Manifest_Handler,
		},
		{
			MethodName: "Run",
			Handler:    _ActionPlugin_Run_Handler,
		},
		{
			MethodName: "WorkerHTTPPort",
			Handler:    _ActionPlugin_WorkerHTTPPort_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _ActionPlugin_Stop_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "actionplugin.proto",
}
