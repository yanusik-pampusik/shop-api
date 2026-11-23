package authclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"shopapi/internal/config"
)

// Connect creates a reusable HTTP/2 connection to the internal gRPC service.
// The deployment keeps this plaintext connection on its private network.
func Connect(cfg *config.Config) (*grpc.ClientConn, error) {
	return grpc.NewClient(cfg.AuthAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
}
