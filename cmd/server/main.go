package main

import (
	"context"
	"log"

	"github.com/waste3d/ghost-tunnel/internal/app"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application, err := app.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	application.Run()
}
