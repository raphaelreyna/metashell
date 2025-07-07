package main

import (
	"context"
	"fmt"

	"github.com/raphaelreyna/metashell/internal/commands"
	"github.com/raphaelreyna/metashell/internal/config"
)

func main() {
	var ctx = context.Background()
	config, err := config.ParseConfig()
	if err != nil {
		panic(fmt.Errorf("error parsing config: %v", err))
	}

	cmd := commands.New(config)
	if err := cmd.Run(ctx); err != nil {
		panic(fmt.Errorf("error running command: %v", err))
	}
}
