package shared

import (
	"context"
	"errors"

	"github.com/raphaelreyna/shelld/pkg/plugin/proto"
)

type DaemonPluginClient struct {
	client proto.DaemonPluginClient
}

func (c *DaemonPluginClient) ReportCommand(ctx context.Context, rep *proto.ReportCommandRequest) error {
	_, err := c.client.ReportCommand(ctx, rep)
	return err
}

func (c *DaemonPluginClient) Metacommand(ctx context.Context, cmd string) (string, error) {
	resp, err := c.client.Metacommand(ctx, &proto.MetacommandRequest{MetaCommand: cmd})
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("got nil response")
	}
	return resp.Out, nil
}

type DaemonPluginServer struct {
	proto.UnimplementedDaemonPluginServer
	Impl DaemonPlugin
}

func (s *DaemonPluginServer) ReportCommand(ctx context.Context, rep *proto.ReportCommandRequest) (*proto.Empty, error) {
	return &proto.Empty{}, s.Impl.ReportCommand(ctx, rep)
}

func (s *DaemonPluginServer) Metacommand(ctx context.Context, req *proto.MetacommandRequest) (*proto.MetacommandResponse, error) {
	out, err := s.Impl.Metacommand(ctx, req.MetaCommand)
	return &proto.MetacommandResponse{Out: out}, err
}
