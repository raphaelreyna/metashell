package plugin

import (
	"context"
	"fmt"
	"net"

	"github.com/raphaelreyna/metashell/internal/config"
	daemonproto "github.com/raphaelreyna/metashell/internal/rpc/go/daemon"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type Cmd struct {
	command *cobra.Command
	config  *config.Config
}

func New(config *config.Config) *Cmd {
	return &Cmd{
		config: config,
	}
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.command != nil {
		return c.command
	}

	c.command = &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
		Long:  "Manage plugins for metashell.",
	}

	c.command.AddCommand(
		c.listCommand(),
	)

	return c.command
}

func (c *Cmd) listCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Long:  "List all plugins known to the metashell daemon.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Connect to the daemon via gRPC (assume Unix socket at c.config.Daemon.SocketPath)
			conn, err := grpc.Dial(
				c.config.Daemon.SocketPath,
				grpc.WithInsecure(),
				grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
					return net.Dial("unix", addr)
				}),
			)
			if err != nil {
				return fmt.Errorf("failed to connect to daemon: %w", err)
			}
			defer conn.Close()

			client := daemonproto.NewMetashellDaemonClient(conn)
			resp, err := client.GetPluginInfo(ctx, &daemonproto.GetPluginInfoRequest{
				PluginName:      "",
				MetacommandName: "",
			})
			if err != nil {
				return fmt.Errorf("failed to get plugin info: %w", err)
			}

			if len(resp.Plugins) == 0 {
				fmt.Println("No plugins found.")
				return nil
			}

			for _, plugin := range resp.Plugins {
				fmt.Printf("Name:        %s\n", plugin.Name)
				fmt.Printf("Version:     %s\n", plugin.Version)
				fmt.Printf("Accepts Command Reports: %v\n", plugin.AcceptsCommandReports)
				if len(plugin.Metacommands) > 0 {
					fmt.Printf("Commands:    ")
					for i, mc := range plugin.Metacommands {
						if i > 0 {
							fmt.Print(", ")
						}
						fmt.Print(mc.Name)
					}
					fmt.Println()
				}
				fmt.Println()
			}

			return nil
		},
	}
}
