package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ghost-tunnel",
	Short: "Ghost Tunnel CLI client",
	Long:  `A client to establish a secure tunnel to the Ghost Tunnel server.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newConnectCmd())
}
