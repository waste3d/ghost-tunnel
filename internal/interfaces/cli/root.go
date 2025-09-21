package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(newConnectCmd())
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newHttpCmd())

	newConnectCmd().Hidden = true
}

func initConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	viper.AddConfigPath(fmt.Sprintf("%s/.config/ghost-tunnel", home))
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.ReadInConfig()
}
