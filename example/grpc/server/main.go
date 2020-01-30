// Copyright 2019, OpenTelemetry Authors
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

package main

import (
	"context"
	"log"
	"net"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/example/grpc/api"
	"go.opentelemetry.io/otel/example/grpc/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/otel/example/grpc/middleware/tracing"
	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	port = ":7777"
)

const (
	maxNest = 1
)

var (
	requests = 0
)

// server is used to implement api.HelloServiceServer
type server struct {
	api.UnimplementedHelloServiceServer
}

// SayHello implements api.HelloServiceServer
func (s *server) SayHello(ctx context.Context, in *api.HelloRequest) (*api.HelloResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	log.Printf("Received: %v, metadata: %v", in.GetGreeting(), md)
	if requests == maxNest {
		return &api.HelloResponse{Reply: "Hello " + in.Greeting}, nil
	}

	requests++

	var conn *grpc.ClientConn
	conn, err := grpc.Dial(port, grpc.WithInsecure(), grpc.WithUnaryInterceptor(tracing.UnaryClientInterceptor))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer func() { _ = conn.Close() }()
	c := api.NewHelloServiceClient(conn)
	clientCtx := metadata.NewOutgoingContext(ctx, md)
	response, err := c.SayHello(clientCtx, &api.HelloRequest{Greeting: "World"})
	if err != nil {
		log.Fatalf("Error when calling SayHello: %s", err)
		return nil, err
	}
	log.Printf("Response from server: %s", response.Reply)
	return response, nil
}

func main() {
	config.Init()

	exporter, err := jaeger.NewExporter(
		jaeger.WithAgentEndpoint("localhost:6831"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "example",
		}))

	if err != nil {
		log.Fatal(err)
	}

	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter),
	)
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)

	defer exporter.Flush()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(tracing.UnaryServerInterceptor))
	go func() {
		time.Sleep(time.Second)
		conn, err := grpc.Dial(port, grpc.WithInsecure(), grpc.WithUnaryInterceptor(tracing.UnaryClientInterceptor))
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer func() { _ = conn.Close() }()
		c := api.NewHelloServiceClient(conn)
		md := metadata.Pairs(
			"user-id", "some-test-user-id",
		)
		clientCtx := metadata.NewOutgoingContext(context.Background(), md)
		response, err := c.SayHello(clientCtx, &api.HelloRequest{Greeting: "World"})
		if err != nil {
			log.Fatalf("Error when calling SayHello: %s", err)
		}
		log.Printf("Response from server: %s", response.Reply)
		s.GracefulStop()
	}()

	api.RegisterHelloServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}