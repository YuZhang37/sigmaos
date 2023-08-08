// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.23.4
// source: cache/proto/cache.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sigmap "sigmaos/sigmap"
	proto "sigmaos/tracing/proto"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type CacheRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key               string                   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value             []byte                   `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Shard             uint32                   `protobuf:"varint,3,opt,name=shard,proto3" json:"shard,omitempty"`
	Mode              uint32                   `protobuf:"varint,4,opt,name=mode,proto3" json:"mode,omitempty"`
	SpanContextConfig *proto.SpanContextConfig `protobuf:"bytes,5,opt,name=spanContextConfig,proto3" json:"spanContextConfig,omitempty"`
	Fence             *sigmap.TfenceProto      `protobuf:"bytes,6,opt,name=fence,proto3" json:"fence,omitempty"`
}

func (x *CacheRequest) Reset() {
	*x = CacheRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheRequest) ProtoMessage() {}

func (x *CacheRequest) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheRequest.ProtoReflect.Descriptor instead.
func (*CacheRequest) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{0}
}

func (x *CacheRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *CacheRequest) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *CacheRequest) GetShard() uint32 {
	if x != nil {
		return x.Shard
	}
	return 0
}

func (x *CacheRequest) GetMode() uint32 {
	if x != nil {
		return x.Mode
	}
	return 0
}

func (x *CacheRequest) GetSpanContextConfig() *proto.SpanContextConfig {
	if x != nil {
		return x.SpanContextConfig
	}
	return nil
}

func (x *CacheRequest) GetFence() *sigmap.TfenceProto {
	if x != nil {
		return x.Fence
	}
	return nil
}

type CacheOK struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *CacheOK) Reset() {
	*x = CacheOK{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheOK) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheOK) ProtoMessage() {}

func (x *CacheOK) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheOK.ProtoReflect.Descriptor instead.
func (*CacheOK) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{1}
}

type CacheResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value []byte `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *CacheResult) Reset() {
	*x = CacheResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheResult) ProtoMessage() {}

func (x *CacheResult) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheResult.ProtoReflect.Descriptor instead.
func (*CacheResult) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{2}
}

func (x *CacheResult) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

type CacheDump struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Vals map[string][]byte `protobuf:"bytes,1,rep,name=vals,proto3" json:"vals,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *CacheDump) Reset() {
	*x = CacheDump{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheDump) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheDump) ProtoMessage() {}

func (x *CacheDump) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheDump.ProtoReflect.Descriptor instead.
func (*CacheDump) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{3}
}

func (x *CacheDump) GetVals() map[string][]byte {
	if x != nil {
		return x.Vals
	}
	return nil
}

type CacheString struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Val string `protobuf:"bytes,1,opt,name=val,proto3" json:"val,omitempty"`
}

func (x *CacheString) Reset() {
	*x = CacheString{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheString) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheString) ProtoMessage() {}

func (x *CacheString) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheString.ProtoReflect.Descriptor instead.
func (*CacheString) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{4}
}

func (x *CacheString) GetVal() string {
	if x != nil {
		return x.Val
	}
	return ""
}

type CacheInt struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Val int64 `protobuf:"varint,1,opt,name=val,proto3" json:"val,omitempty"`
}

func (x *CacheInt) Reset() {
	*x = CacheInt{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheInt) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheInt) ProtoMessage() {}

func (x *CacheInt) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheInt.ProtoReflect.Descriptor instead.
func (*CacheInt) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{5}
}

func (x *CacheInt) GetVal() int64 {
	if x != nil {
		return x.Val
	}
	return 0
}

type ShardArg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Shard uint32              `protobuf:"varint,1,opt,name=shard,proto3" json:"shard,omitempty"`
	Fence *sigmap.TfenceProto `protobuf:"bytes,2,opt,name=fence,proto3" json:"fence,omitempty"`
	Vals  map[string][]byte   `protobuf:"bytes,3,rep,name=vals,proto3" json:"vals,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *ShardArg) Reset() {
	*x = ShardArg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShardArg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShardArg) ProtoMessage() {}

func (x *ShardArg) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShardArg.ProtoReflect.Descriptor instead.
func (*ShardArg) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{6}
}

func (x *ShardArg) GetShard() uint32 {
	if x != nil {
		return x.Shard
	}
	return 0
}

func (x *ShardArg) GetFence() *sigmap.TfenceProto {
	if x != nil {
		return x.Fence
	}
	return nil
}

func (x *ShardArg) GetVals() map[string][]byte {
	if x != nil {
		return x.Vals
	}
	return nil
}

type ShardFill struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Shard uint32            `protobuf:"varint,1,opt,name=shard,proto3" json:"shard,omitempty"`
	Vals  map[string][]byte `protobuf:"bytes,2,rep,name=vals,proto3" json:"vals,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *ShardFill) Reset() {
	*x = ShardFill{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cache_proto_cache_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShardFill) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShardFill) ProtoMessage() {}

func (x *ShardFill) ProtoReflect() protoreflect.Message {
	mi := &file_cache_proto_cache_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShardFill.ProtoReflect.Descriptor instead.
func (*ShardFill) Descriptor() ([]byte, []int) {
	return file_cache_proto_cache_proto_rawDescGZIP(), []int{7}
}

func (x *ShardFill) GetShard() uint32 {
	if x != nil {
		return x.Shard
	}
	return 0
}

func (x *ShardFill) GetVals() map[string][]byte {
	if x != nil {
		return x.Vals
	}
	return nil
}

var File_cache_proto_cache_proto protoreflect.FileDescriptor

var file_cache_proto_cache_proto_rawDesc = []byte{
	0x0a, 0x17, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x61,
	0x63, 0x68, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x74, 0x72, 0x61, 0x63, 0x69,
	0x6e, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x69, 0x6e, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x13, 0x73, 0x69, 0x67, 0x6d, 0x61, 0x70, 0x2f, 0x73,
	0x69, 0x67, 0x6d, 0x61, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc6, 0x01, 0x0a, 0x0c,
	0x43, 0x61, 0x63, 0x68, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x68, 0x61, 0x72, 0x64, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x05, 0x73, 0x68, 0x61, 0x72, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6d, 0x6f,
	0x64, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x6d, 0x6f, 0x64, 0x65, 0x12, 0x40,
	0x0a, 0x11, 0x73, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x53, 0x70, 0x61, 0x6e,
	0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x11, 0x73,
	0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x12, 0x22, 0x0a, 0x05, 0x66, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x0c, 0x2e, 0x54, 0x66, 0x65, 0x6e, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x05, 0x66,
	0x65, 0x6e, 0x63, 0x65, 0x22, 0x09, 0x0a, 0x07, 0x43, 0x61, 0x63, 0x68, 0x65, 0x4f, 0x4b, 0x22,
	0x23, 0x0a, 0x0b, 0x43, 0x61, 0x63, 0x68, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x22, 0x6e, 0x0a, 0x09, 0x43, 0x61, 0x63, 0x68, 0x65, 0x44, 0x75, 0x6d,
	0x70, 0x12, 0x28, 0x0a, 0x04, 0x76, 0x61, 0x6c, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x14, 0x2e, 0x43, 0x61, 0x63, 0x68, 0x65, 0x44, 0x75, 0x6d, 0x70, 0x2e, 0x56, 0x61, 0x6c, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x76, 0x61, 0x6c, 0x73, 0x1a, 0x37, 0x0a, 0x09, 0x56,
	0x61, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x22, 0x1f, 0x0a, 0x0b, 0x43, 0x61, 0x63, 0x68, 0x65, 0x53, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x12, 0x10, 0x0a, 0x03, 0x76, 0x61, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x76, 0x61, 0x6c, 0x22, 0x1c, 0x0a, 0x08, 0x43, 0x61, 0x63, 0x68, 0x65, 0x49, 0x6e,
	0x74, 0x12, 0x10, 0x0a, 0x03, 0x76, 0x61, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03,
	0x76, 0x61, 0x6c, 0x22, 0xa6, 0x01, 0x0a, 0x08, 0x53, 0x68, 0x61, 0x72, 0x64, 0x41, 0x72, 0x67,
	0x12, 0x14, 0x0a, 0x05, 0x73, 0x68, 0x61, 0x72, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x05, 0x73, 0x68, 0x61, 0x72, 0x64, 0x12, 0x22, 0x0a, 0x05, 0x66, 0x65, 0x6e, 0x63, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x54, 0x66, 0x65, 0x6e, 0x63, 0x65, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x52, 0x05, 0x66, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x27, 0x0a, 0x04, 0x76, 0x61,
	0x6c, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x53, 0x68, 0x61, 0x72, 0x64,
	0x41, 0x72, 0x67, 0x2e, 0x56, 0x61, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x76,
	0x61, 0x6c, 0x73, 0x1a, 0x37, 0x0a, 0x09, 0x56, 0x61, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x84, 0x01, 0x0a,
	0x09, 0x53, 0x68, 0x61, 0x72, 0x64, 0x46, 0x69, 0x6c, 0x6c, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x68,
	0x61, 0x72, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x73, 0x68, 0x61, 0x72, 0x64,
	0x12, 0x28, 0x0a, 0x04, 0x76, 0x61, 0x6c, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x14,
	0x2e, 0x53, 0x68, 0x61, 0x72, 0x64, 0x46, 0x69, 0x6c, 0x6c, 0x2e, 0x56, 0x61, 0x6c, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x76, 0x61, 0x6c, 0x73, 0x1a, 0x37, 0x0a, 0x09, 0x56, 0x61,
	0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x32, 0x76, 0x0a, 0x05, 0x43, 0x61, 0x63, 0x68, 0x65, 0x12, 0x22, 0x0a, 0x03,
	0x47, 0x65, 0x74, 0x12, 0x0d, 0x2e, 0x43, 0x61, 0x63, 0x68, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x0c, 0x2e, 0x43, 0x61, 0x63, 0x68, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74,
	0x12, 0x22, 0x0a, 0x03, 0x53, 0x65, 0x74, 0x12, 0x0d, 0x2e, 0x43, 0x61, 0x63, 0x68, 0x65, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0c, 0x2e, 0x43, 0x61, 0x63, 0x68, 0x65, 0x52, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x12, 0x25, 0x0a, 0x06, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x0d,
	0x2e, 0x43, 0x61, 0x63, 0x68, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0c, 0x2e,
	0x43, 0x61, 0x63, 0x68, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x42, 0x15, 0x5a, 0x13, 0x73,
	0x69, 0x67, 0x6d, 0x61, 0x6f, 0x73, 0x2f, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cache_proto_cache_proto_rawDescOnce sync.Once
	file_cache_proto_cache_proto_rawDescData = file_cache_proto_cache_proto_rawDesc
)

func file_cache_proto_cache_proto_rawDescGZIP() []byte {
	file_cache_proto_cache_proto_rawDescOnce.Do(func() {
		file_cache_proto_cache_proto_rawDescData = protoimpl.X.CompressGZIP(file_cache_proto_cache_proto_rawDescData)
	})
	return file_cache_proto_cache_proto_rawDescData
}

var file_cache_proto_cache_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_cache_proto_cache_proto_goTypes = []interface{}{
	(*CacheRequest)(nil),            // 0: CacheRequest
	(*CacheOK)(nil),                 // 1: CacheOK
	(*CacheResult)(nil),             // 2: CacheResult
	(*CacheDump)(nil),               // 3: CacheDump
	(*CacheString)(nil),             // 4: CacheString
	(*CacheInt)(nil),                // 5: CacheInt
	(*ShardArg)(nil),                // 6: ShardArg
	(*ShardFill)(nil),               // 7: ShardFill
	nil,                             // 8: CacheDump.ValsEntry
	nil,                             // 9: ShardArg.ValsEntry
	nil,                             // 10: ShardFill.ValsEntry
	(*proto.SpanContextConfig)(nil), // 11: SpanContextConfig
	(*sigmap.TfenceProto)(nil),      // 12: TfenceProto
}
var file_cache_proto_cache_proto_depIdxs = []int32{
	11, // 0: CacheRequest.spanContextConfig:type_name -> SpanContextConfig
	12, // 1: CacheRequest.fence:type_name -> TfenceProto
	8,  // 2: CacheDump.vals:type_name -> CacheDump.ValsEntry
	12, // 3: ShardArg.fence:type_name -> TfenceProto
	9,  // 4: ShardArg.vals:type_name -> ShardArg.ValsEntry
	10, // 5: ShardFill.vals:type_name -> ShardFill.ValsEntry
	0,  // 6: Cache.Get:input_type -> CacheRequest
	0,  // 7: Cache.Set:input_type -> CacheRequest
	0,  // 8: Cache.Delete:input_type -> CacheRequest
	2,  // 9: Cache.Get:output_type -> CacheResult
	2,  // 10: Cache.Set:output_type -> CacheResult
	2,  // 11: Cache.Delete:output_type -> CacheResult
	9,  // [9:12] is the sub-list for method output_type
	6,  // [6:9] is the sub-list for method input_type
	6,  // [6:6] is the sub-list for extension type_name
	6,  // [6:6] is the sub-list for extension extendee
	0,  // [0:6] is the sub-list for field type_name
}

func init() { file_cache_proto_cache_proto_init() }
func file_cache_proto_cache_proto_init() {
	if File_cache_proto_cache_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cache_proto_cache_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheRequest); i {
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
		file_cache_proto_cache_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheOK); i {
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
		file_cache_proto_cache_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheResult); i {
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
		file_cache_proto_cache_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheDump); i {
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
		file_cache_proto_cache_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheString); i {
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
		file_cache_proto_cache_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheInt); i {
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
		file_cache_proto_cache_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ShardArg); i {
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
		file_cache_proto_cache_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ShardFill); i {
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
			RawDescriptor: file_cache_proto_cache_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_cache_proto_cache_proto_goTypes,
		DependencyIndexes: file_cache_proto_cache_proto_depIdxs,
		MessageInfos:      file_cache_proto_cache_proto_msgTypes,
	}.Build()
	File_cache_proto_cache_proto = out.File
	file_cache_proto_cache_proto_rawDesc = nil
	file_cache_proto_cache_proto_goTypes = nil
	file_cache_proto_cache_proto_depIdxs = nil
}
