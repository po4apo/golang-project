package client

import (
	"fmt"

	authv1 "golang-project/api/proto/gen/go/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthClient представляет gRPC клиент для auth-service
type AuthClient struct {
	conn   *grpc.ClientConn
	Client authv1.AuthServiceClient
}

// NewAuthClient создаёт новый gRPC клиент для auth-service
func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	return &AuthClient{
		conn:   conn,
		Client: authv1.NewAuthServiceClient(conn),
	}, nil
}

// Close закрывает соединение с gRPC сервером
func (c *AuthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}







