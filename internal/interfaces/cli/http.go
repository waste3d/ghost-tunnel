package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newHttpCmd() *cobra.Command {
	var serverAPI, serverGRPC, subdomain string

	cmd := &cobra.Command{
		Use:   "http [port]",
		Short: "Create a new HTTP tunnel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			localPort, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid port number: %s", args[0])
			}

			// 1. Загружаем API-ключ из конфига
			apiKey := viper.GetString("api_key")
			if apiKey == "" {
				return fmt.Errorf("not logged in. Please run 'ghost-tunnel login' first")
			}

			// 2. Делаем запрос на создание туннеля
			reqBody, _ := json.Marshal(map[string]interface{}{
				"subdomain": subdomain, // Будет пустым, если не указан флаг
				"localport": localPort,
			})

			client := &http.Client{}
			req, _ := http.NewRequest("POST", fmt.Sprintf("%s/tunnels", serverAPI), bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("could not create tunnel: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				bodyBytes, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed to create tunnel (status %d): %s", resp.StatusCode, string(bodyBytes))
			}

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			tunnelID := result["ID"].(string)
			endpoint := result["Endpoints"].(map[string]interface{})
			publicSubdomain := endpoint["Subdomain"].(string)
			publicDomain := endpoint["Domain"].(string)

			// 3. Выводим красивый URL
			log.Printf("Tunnel created successfully!")
			log.Printf("Forwarding http://%s.%s -> http://localhost:%d", publicSubdomain, publicDomain, localPort)

			// 4. Запускаем gRPC-клиент с полученным ID
			tunnelClient := NewClient(tunnelID, fmt.Sprintf("localhost:%d", localPort))
			return tunnelClient.Run(cmd.Context(), serverGRPC)
		},
	}

	cmd.Flags().StringVar(&serverAPI, "api-server", "https://api.gtunnel.ru", "The address of the API server")
	cmd.Flags().StringVar(&serverGRPC, "grpc-server", "83.166.247.105:50051", "The address of the gRPC server")
	cmd.Flags().StringVar(&subdomain, "subdomain", "", "Request a specific subdomain (if available)")
	return cmd
}
