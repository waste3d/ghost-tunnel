package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func newLoginCmd() *cobra.Command {
	var serverAPI string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Ghost Tunnel server and save the API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Enter Email: ")
			var email string
			fmt.Scanln(&email)

			fmt.Print("Enter Password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("could not read password: %w", err)
			}
			password := string(bytePassword)
			fmt.Println() // Для перевода строки после скрытого ввода

			// Формируем тело запроса
			reqBody, _ := json.Marshal(map[string]string{
				"email":    email,
				"password": password,
			})

			// Делаем HTTP-запрос к API
			resp, err := http.Post(fmt.Sprintf("%s/login", serverAPI), "application/json", bytes.NewBuffer(reqBody))
			if err != nil {
				return fmt.Errorf("could not connect to server: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(bodyBytes))
			}

			// Парсим ответ
			var result map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("could not parse server response: %w", err)
			}

			apiKey, ok := result["api_key"]
			if !ok {
				return fmt.Errorf("API key not found in server response")
			}

			// Сохраняем ключ в конфигурационный файл
			viper.Set("api_key", apiKey)

			// Находим домашнюю директорию
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			// Создаем директорию .config/ghost-tunnel, если ее нет
			configPath := fmt.Sprintf("%s/.config/ghost-tunnel", home)
			if err := os.MkdirAll(configPath, 0755); err != nil {
				return err
			}

			// Пытаемся создать или перезаписать файл конфигурации
			if err := viper.WriteConfigAs(fmt.Sprintf("%s/config.yaml", configPath)); err != nil {
				return fmt.Errorf("could not save config file: %w", err)
			}

			log.Println("Login successful! API key saved.")
			return nil
		},
	}

	cmd.Flags().StringVar(&serverAPI, "api-server", "https://api.gtunnel.ru", "The address of the API server")
	return cmd
}
