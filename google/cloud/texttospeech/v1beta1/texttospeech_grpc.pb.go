// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.20.3
// source: google/cloud/texttospeech/v1beta1/texttospeech.proto

package texttospeechpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	TextToSpeech_ListVoices_FullMethodName          = "/google.cloud.texttospeech.v1beta1.TextToSpeech/ListVoices"
	TextToSpeech_SynthesizeSpeech_FullMethodName    = "/google.cloud.texttospeech.v1beta1.TextToSpeech/SynthesizeSpeech"
	TextToSpeech_StreamingSynthesize_FullMethodName = "/google.cloud.texttospeech.v1beta1.TextToSpeech/StreamingSynthesize"
)

// TextToSpeechClient is the client API for TextToSpeech service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Service that implements Google Cloud Text-to-Speech API.
type TextToSpeechClient interface {
	// Returns a list of Voice supported for synthesis.
	ListVoices(ctx context.Context, in *ListVoicesRequest, opts ...grpc.CallOption) (*ListVoicesResponse, error)
	// Synthesizes speech synchronously: receive results after all text input
	// has been processed.
	SynthesizeSpeech(ctx context.Context, in *SynthesizeSpeechRequest, opts ...grpc.CallOption) (*SynthesizeSpeechResponse, error)
	// Performs bidirectional streaming speech synthesis: receive audio while
	// sending text.
	StreamingSynthesize(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[StreamingSynthesizeRequest, StreamingSynthesizeResponse], error)
}

type textToSpeechClient struct {
	cc grpc.ClientConnInterface
}

func NewTextToSpeechClient(cc grpc.ClientConnInterface) TextToSpeechClient {
	return &textToSpeechClient{cc}
}

func (c *textToSpeechClient) ListVoices(ctx context.Context, in *ListVoicesRequest, opts ...grpc.CallOption) (*ListVoicesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListVoicesResponse)
	err := c.cc.Invoke(ctx, TextToSpeech_ListVoices_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *textToSpeechClient) SynthesizeSpeech(ctx context.Context, in *SynthesizeSpeechRequest, opts ...grpc.CallOption) (*SynthesizeSpeechResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SynthesizeSpeechResponse)
	err := c.cc.Invoke(ctx, TextToSpeech_SynthesizeSpeech_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *textToSpeechClient) StreamingSynthesize(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[StreamingSynthesizeRequest, StreamingSynthesizeResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &TextToSpeech_ServiceDesc.Streams[0], TextToSpeech_StreamingSynthesize_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[StreamingSynthesizeRequest, StreamingSynthesizeResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type TextToSpeech_StreamingSynthesizeClient = grpc.BidiStreamingClient[StreamingSynthesizeRequest, StreamingSynthesizeResponse]

// TextToSpeechServer is the server API for TextToSpeech service.
// All implementations must embed UnimplementedTextToSpeechServer
// for forward compatibility.
//
// Service that implements Google Cloud Text-to-Speech API.
type TextToSpeechServer interface {
	// Returns a list of Voice supported for synthesis.
	ListVoices(context.Context, *ListVoicesRequest) (*ListVoicesResponse, error)
	// Synthesizes speech synchronously: receive results after all text input
	// has been processed.
	SynthesizeSpeech(context.Context, *SynthesizeSpeechRequest) (*SynthesizeSpeechResponse, error)
	// Performs bidirectional streaming speech synthesis: receive audio while
	// sending text.
	StreamingSynthesize(grpc.BidiStreamingServer[StreamingSynthesizeRequest, StreamingSynthesizeResponse]) error
	mustEmbedUnimplementedTextToSpeechServer()
}

// UnimplementedTextToSpeechServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedTextToSpeechServer struct{}

func (UnimplementedTextToSpeechServer) ListVoices(context.Context, *ListVoicesRequest) (*ListVoicesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListVoices not implemented")
}
func (UnimplementedTextToSpeechServer) SynthesizeSpeech(context.Context, *SynthesizeSpeechRequest) (*SynthesizeSpeechResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SynthesizeSpeech not implemented")
}
func (UnimplementedTextToSpeechServer) StreamingSynthesize(grpc.BidiStreamingServer[StreamingSynthesizeRequest, StreamingSynthesizeResponse]) error {
	return status.Errorf(codes.Unimplemented, "method StreamingSynthesize not implemented")
}
func (UnimplementedTextToSpeechServer) mustEmbedUnimplementedTextToSpeechServer() {}
func (UnimplementedTextToSpeechServer) testEmbeddedByValue()                      {}

// UnsafeTextToSpeechServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TextToSpeechServer will
// result in compilation errors.
type UnsafeTextToSpeechServer interface {
	mustEmbedUnimplementedTextToSpeechServer()
}

func RegisterTextToSpeechServer(s grpc.ServiceRegistrar, srv TextToSpeechServer) {
	// If the following call pancis, it indicates UnimplementedTextToSpeechServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&TextToSpeech_ServiceDesc, srv)
}

func _TextToSpeech_ListVoices_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListVoicesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TextToSpeechServer).ListVoices(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TextToSpeech_ListVoices_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TextToSpeechServer).ListVoices(ctx, req.(*ListVoicesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TextToSpeech_SynthesizeSpeech_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SynthesizeSpeechRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TextToSpeechServer).SynthesizeSpeech(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TextToSpeech_SynthesizeSpeech_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TextToSpeechServer).SynthesizeSpeech(ctx, req.(*SynthesizeSpeechRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TextToSpeech_StreamingSynthesize_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(TextToSpeechServer).StreamingSynthesize(&grpc.GenericServerStream[StreamingSynthesizeRequest, StreamingSynthesizeResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type TextToSpeech_StreamingSynthesizeServer = grpc.BidiStreamingServer[StreamingSynthesizeRequest, StreamingSynthesizeResponse]

// TextToSpeech_ServiceDesc is the grpc.ServiceDesc for TextToSpeech service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TextToSpeech_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "google.cloud.texttospeech.v1beta1.TextToSpeech",
	HandlerType: (*TextToSpeechServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListVoices",
			Handler:    _TextToSpeech_ListVoices_Handler,
		},
		{
			MethodName: "SynthesizeSpeech",
			Handler:    _TextToSpeech_SynthesizeSpeech_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamingSynthesize",
			Handler:       _TextToSpeech_StreamingSynthesize_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "google/cloud/texttospeech/v1beta1/texttospeech.proto",
}