package cli

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

func newConnectCmd() *cobra.Command {
	var serverAddr string
	var tunnelID string
	var localAddr string

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to the server and start a tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			client := NewClient(tunnelID, localAddr)
			if err := client.Run(context.Background(), serverAddr); err != nil {
				log.Fatalf("Client error: %v", err)
			}
		},
	}

	// Определяем флаги для команды
	cmd.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "Server address")
	cmd.Flags().StringVarP(&tunnelID, "tunnel-id", "t", "", "Tunnel ID to connect to")
	cmd.Flags().StringVarP(&localAddr, "local", "l", "localhost:8080", "Local address to forward traffic to (host:port)")

	_ = cmd.MarkFlagRequired("tunnel-id")

	return cmd
}
