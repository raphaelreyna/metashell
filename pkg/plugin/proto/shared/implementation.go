package shared

import (
	"context"

	"github.com/raphaelreyna/shelld/pkg/plugin/proto"
)

type CommandReportHandlerClient struct {
	client proto.CommandReportHandlerClient
}

func (c *CommandReportHandlerClient) ReportCommand(ctx context.Context, rep *proto.CommandReport) error {
	_, err := c.client.ReportCommand(ctx, rep)
	return err
}

type CommandReportHandlerServer struct {
	proto.UnimplementedCommandReportHandlerServer
	Impl CommandReportHandler
}

func (s *CommandReportHandlerServer) ReportCommand(ctx context.Context, rep *proto.CommandReport) (*proto.Empty, error) {
	return &proto.Empty{}, s.Impl.ReportCommand(ctx, rep)
}
