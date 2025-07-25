package shared

import (
	"context"
	"errors"

	"github.com/raphaelreyna/metashell/pkg/plugin/proto/proto"
)

type DaemonPluginClient struct {
	client proto.DaemonPluginClient
}

func (c *DaemonPluginClient) ReportCommand(ctx context.Context, rep *proto.ReportCommandRequest) error {
	_, err := c.client.ReportCommand(ctx, rep)
	return err
}

func (c *DaemonPluginClient) Metacommand(ctx context.Context, req *proto.MetacommandRequest) (*proto.MetacommandResponse, error) {
	resp, err := c.client.Metacommand(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("got nil response")
	}
	return resp, nil
}

func (c *DaemonPluginClient) Info(ctx context.Context) (*proto.PluginInfo, error) {
	return c.client.Info(ctx, &proto.Empty{})
}

func (c *DaemonPluginClient) Init(ctx context.Context, config *proto.PluginConfig) error {
	_, err := c.client.Init(ctx, config)
	return err
}

type DaemonPluginServer struct {
	proto.UnimplementedDaemonPluginServer
	Impl DaemonPlugin
}

func (s *DaemonPluginServer) ReportCommand(ctx context.Context, rep *proto.ReportCommandRequest) (*proto.Empty, error) {
	return &proto.Empty{}, s.Impl.ReportCommand(ctx, rep)
}

func (s *DaemonPluginServer) Metacommand(ctx context.Context, req *proto.MetacommandRequest) (*proto.MetacommandResponse, error) {
	resp, err := s.Impl.Metacommand(ctx, req)
	if err != nil {
		resp.Error = err.Error()
	}

	return resp, err
}

func (s *DaemonPluginServer) Info(ctx context.Context, _ *proto.Empty) (*proto.PluginInfo, error) {
	return s.Impl.Info(ctx)
}

func (s *DaemonPluginServer) Init(ctx context.Context, config *proto.PluginConfig) (*proto.Empty, error) {
	if err := s.Impl.Init(ctx, config); err != nil {
		return &proto.Empty{}, err
	}
	return &proto.Empty{}, nil
}
