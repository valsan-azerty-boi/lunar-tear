package interceptor

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Logging(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	log.Printf(">>> %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("<<< %s ERROR: %v", info.FullMethod, err)
	} else {
		log.Printf("<<< %s OK", info.FullMethod)
	}
	return resp, err
}

func UnknownService(_ any, stream grpc.ServerStream) error {
	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		fullMethod = "<unknown>"
	}
	log.Printf(">>> %s", fullMethod)
	err := status.Errorf(codes.Unimplemented, "unknown service or method %s", fullMethod)
	log.Printf("<<< %s ERROR: %v", fullMethod, err)
	return err
}
