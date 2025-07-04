package client

import (
	"context"
	"fmt"
	"time"

	shortenerv1 "GURLS-Redirect/gen/go/shortener/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BackendClient struct {
	conn   *grpc.ClientConn
	client shortenerv1.ShortenerClient
	log    *zap.Logger
}

func NewBackendClient(address string, timeout time.Duration, log *zap.Logger) (*BackendClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to backend: %w", err)
	}

	client := shortenerv1.NewShortenerClient(conn)

	return &BackendClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *BackendClient) GetLink(ctx context.Context, alias string) (*shortenerv1.GetLinkStatsResponse, error) {
	req := &shortenerv1.GetLinkStatsRequest{Alias: alias}
	resp, err := c.client.GetLinkStats(ctx, req)
	if err != nil {
		c.log.Error("failed to get link from backend", zap.Error(err), zap.String("alias", alias))
		return nil, err
	}
	return resp, nil
}

func (c *BackendClient) RecordClick(ctx context.Context, alias, deviceType string) error {
	req := &shortenerv1.RecordClickRequest{
		Alias:      alias,
		DeviceType: deviceType,
	}
	_, err := c.client.RecordClick(ctx, req)
	if err != nil {
		c.log.Error("failed to record click via backend", zap.Error(err), zap.String("alias", alias))
		return err
	}
	return nil
}

func (c *BackendClient) Close() error {
	return c.conn.Close()
}