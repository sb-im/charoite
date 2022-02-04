// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.6.1
// source: signal.proto

package signal

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type TrackSource int32

const (
	TrackSource_UNKNOWN TrackSource = 0
	TrackSource_DRONE   TrackSource = 1
	TrackSource_MONITOR TrackSource = 2
)

// Enum value maps for TrackSource.
var (
	TrackSource_name = map[int32]string{
		0: "UNKNOWN",
		1: "DRONE",
		2: "MONITOR",
	}
	TrackSource_value = map[string]int32{
		"UNKNOWN": 0,
		"DRONE":   1,
		"MONITOR": 2,
	}
)

func (x TrackSource) Enum() *TrackSource {
	p := new(TrackSource)
	*p = x
	return p
}

func (x TrackSource) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (TrackSource) Descriptor() protoreflect.EnumDescriptor {
	return file_signal_proto_enumTypes[0].Descriptor()
}

func (TrackSource) Type() protoreflect.EnumType {
	return &file_signal_proto_enumTypes[0]
}

func (x TrackSource) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use TrackSource.Descriptor instead.
func (TrackSource) EnumDescriptor() ([]byte, []int) {
	return file_signal_proto_rawDescGZIP(), []int{0}
}

type SessionDescription struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Meta *Meta  `protobuf:"bytes,1,opt,name=meta,proto3" json:"meta,omitempty"` // Metadata to identify actor if any
	Sdp  string `protobuf:"bytes,2,opt,name=sdp,proto3" json:"sdp,omitempty"`   // JSON encoded webrtc.SessionDescription
}

func (x *SessionDescription) Reset() {
	*x = SessionDescription{}
	if protoimpl.UnsafeEnabled {
		mi := &file_signal_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SessionDescription) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SessionDescription) ProtoMessage() {}

func (x *SessionDescription) ProtoReflect() protoreflect.Message {
	mi := &file_signal_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SessionDescription.ProtoReflect.Descriptor instead.
func (*SessionDescription) Descriptor() ([]byte, []int) {
	return file_signal_proto_rawDescGZIP(), []int{0}
}

func (x *SessionDescription) GetMeta() *Meta {
	if x != nil {
		return x.Meta
	}
	return nil
}

func (x *SessionDescription) GetSdp() string {
	if x != nil {
		return x.Sdp
	}
	return ""
}

type ICECandidate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Meta      *Meta  `protobuf:"bytes,1,opt,name=meta,proto3" json:"meta,omitempty"`           // Metadata to identify actor if any
	Candidate string `protobuf:"bytes,2,opt,name=candidate,proto3" json:"candidate,omitempty"` // JSON encoded webrtc.ICECandidate.
}

func (x *ICECandidate) Reset() {
	*x = ICECandidate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_signal_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ICECandidate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ICECandidate) ProtoMessage() {}

func (x *ICECandidate) ProtoReflect() protoreflect.Message {
	mi := &file_signal_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ICECandidate.ProtoReflect.Descriptor instead.
func (*ICECandidate) Descriptor() ([]byte, []int) {
	return file_signal_proto_rawDescGZIP(), []int{1}
}

func (x *ICECandidate) GetMeta() *Meta {
	if x != nil {
		return x.Meta
	}
	return nil
}

func (x *ICECandidate) GetCandidate() string {
	if x != nil {
		return x.Candidate
	}
	return ""
}

type Meta struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id          string      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"` // Unique machine ID for edge device.
	TrackSource TrackSource `protobuf:"varint,2,opt,name=track_source,json=trackSource,proto3,enum=pb.TrackSource" json:"track_source,omitempty"`
}

func (x *Meta) Reset() {
	*x = Meta{}
	if protoimpl.UnsafeEnabled {
		mi := &file_signal_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Meta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Meta) ProtoMessage() {}

func (x *Meta) ProtoReflect() protoreflect.Message {
	mi := &file_signal_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Meta.ProtoReflect.Descriptor instead.
func (*Meta) Descriptor() ([]byte, []int) {
	return file_signal_proto_rawDescGZIP(), []int{2}
}

func (x *Meta) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Meta) GetTrackSource() TrackSource {
	if x != nil {
		return x.TrackSource
	}
	return TrackSource_UNKNOWN
}

var File_signal_proto protoreflect.FileDescriptor

var file_signal_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02,
	0x70, 0x62, 0x22, 0x44, 0x0a, 0x12, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x44, 0x65, 0x73,
	0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1c, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x65, 0x74, 0x61,
	0x52, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x64, 0x70, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x73, 0x64, 0x70, 0x22, 0x4a, 0x0a, 0x0c, 0x49, 0x43, 0x45, 0x43,
	0x61, 0x6e, 0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x12, 0x1c, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x65, 0x74, 0x61,
	0x52, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x12, 0x1c, 0x0a, 0x09, 0x63, 0x61, 0x6e, 0x64, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63, 0x61, 0x6e, 0x64, 0x69,
	0x64, 0x61, 0x74, 0x65, 0x22, 0x4a, 0x0a, 0x04, 0x4d, 0x65, 0x74, 0x61, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x32, 0x0a, 0x0c,
	0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x53, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x52, 0x0b, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x2a, 0x32, 0x0a, 0x0b, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12,
	0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05,
	0x44, 0x52, 0x4f, 0x4e, 0x45, 0x10, 0x01, 0x12, 0x0b, 0x0a, 0x07, 0x4d, 0x4f, 0x4e, 0x49, 0x54,
	0x4f, 0x52, 0x10, 0x02, 0x42, 0x1c, 0x5a, 0x1a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x53, 0x42, 0x2d, 0x49, 0x4d, 0x2f, 0x70, 0x62, 0x2f, 0x73, 0x69, 0x67, 0x6e,
	0x61, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_signal_proto_rawDescOnce sync.Once
	file_signal_proto_rawDescData = file_signal_proto_rawDesc
)

func file_signal_proto_rawDescGZIP() []byte {
	file_signal_proto_rawDescOnce.Do(func() {
		file_signal_proto_rawDescData = protoimpl.X.CompressGZIP(file_signal_proto_rawDescData)
	})
	return file_signal_proto_rawDescData
}

var file_signal_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_signal_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_signal_proto_goTypes = []interface{}{
	(TrackSource)(0),           // 0: pb.TrackSource
	(*SessionDescription)(nil), // 1: pb.SessionDescription
	(*ICECandidate)(nil),       // 2: pb.ICECandidate
	(*Meta)(nil),               // 3: pb.Meta
}
var file_signal_proto_depIdxs = []int32{
	3, // 0: pb.SessionDescription.meta:type_name -> pb.Meta
	3, // 1: pb.ICECandidate.meta:type_name -> pb.Meta
	0, // 2: pb.Meta.track_source:type_name -> pb.TrackSource
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_signal_proto_init() }
func file_signal_proto_init() {
	if File_signal_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_signal_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SessionDescription); i {
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
		file_signal_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ICECandidate); i {
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
		file_signal_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Meta); i {
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
			RawDescriptor: file_signal_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_signal_proto_goTypes,
		DependencyIndexes: file_signal_proto_depIdxs,
		EnumInfos:         file_signal_proto_enumTypes,
		MessageInfos:      file_signal_proto_msgTypes,
	}.Build()
	File_signal_proto = out.File
	file_signal_proto_rawDesc = nil
	file_signal_proto_goTypes = nil
	file_signal_proto_depIdxs = nil
}