package main

import (
	"context"
	"fmt"

	"github.com/raphaelreyna/shelld/metashell"
)

func main() {
	var (
		ctx    = context.Background()
		bundle = metashell.InitBundle{
			RootDir: "/home/rr/shwr/_test",
			Shell:   "/bin/bash",
			PostRunReportHandlerFunc: func(ctx context.Context, prr *metashell.PostRunReport) error {
				out := metashell.GetStdout(ctx)
				fmt.Fprintf(out, "PostRunReport: %+v\n", *prr)
				return nil
			},
		}
	)

	if err := metashell.Init(ctx, &bundle); err != nil {
		panic(err)
	}
}
