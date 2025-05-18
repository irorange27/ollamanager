// ollamanager - A wrapper for ollama that allows controlling ollama instances
// available on your internal network. With ollamanager, you can manage multiple
// ollama servers and seamlessly switch between them.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Configuration types
type OllamaServer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type Config struct {
	Servers    []OllamaServer `json:"servers"`
	Current    string         `json:"current"`
	ConfigPath string         `json:"-"`
}

// Global variables
var config Config
var version = "0.1.0"
var appName = "ollamanager"
var description = "A wrapper for ollama that allows controlling ollama instances on your internal network"

// Main entry point
func main() {
	// Initialize configuration
	initConfig()

	// Check parameters
	if len(os.Args) < 2 {
		printShortHelp()
		return
	}

	command := os.Args[1]

	// Handle command routing
	switch command {
	case "server":
		handleServerCommand()
	case "help":
		handleHelpCommand()
	case "version":
		handleVersionCommand()
	default:
		// Pass command directly to ollama
		runOllamaCommand(command, os.Args[2:])
	}
}

// Command handlers
func handleServerCommand() {
	if len(os.Args) < 3 {
		printServerHelp()
		return
	}

	subCommand := os.Args[2]
	args := os.Args[3:]

	switch subCommand {
	case "add":
		if len(args) < 2 {
			fmt.Println("Usage: ollamanager server add <name> <address>")
			return
		}
		addServer(args[0], args[1])
	case "list":
		listServers()
	case "use":
		if len(args) < 1 {
			fmt.Println("Usage: ollamanager server use <name>")
			return
		}
		useServer(args[0])
	case "remove":
		if len(args) < 1 {
			fmt.Println("Usage: ollamanager server remove <name>")
			return
		}
		removeServer(args[0])
	case "current":
		showCurrentServer()
	case "ping":
		pingCurrentServer()
	default:
		fmt.Printf("Unknown server command: %s\n", subCommand)
		printServerHelp()
	}
}

func handleHelpCommand() {
	if len(os.Args) > 2 && os.Args[2] == "server" {
		printServerHelp()
	} else {
		printDetailedHelp()
	}
}

func handleVersionCommand() {
	fmt.Printf("Ollamanager v%s\n", version)
	server := getCurrentServer()
	if server != nil {
		os.Setenv("OLLAMA_HOST", server.Address)
		cmd := exec.Command("ollama", "--version")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

// Configuration management functions
func initConfig() {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Printf("Warning: Could not determine config directory: %v\n", err)
		configDir = "."
	}

	// Create config directory
	ollamaDir := filepath.Join(configDir, "ollamanager")
	if err := os.MkdirAll(ollamaDir, 0755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Config file path
	configPath := filepath.Join(ollamaDir, "config.json")
	config.ConfigPath = configPath

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config = Config{
			Servers: []OllamaServer{
				{Name: "default", Address: "127.0.0.1:11434"},
			},
			Current:    "default",
			ConfigPath: configPath,
		}
		if err := saveConfig(); err != nil {
			fmt.Printf("Error creating default config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created default config at %s\n", configPath)
		return
	}

	// Read existing config
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("Error opening config file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		// Backup corrupted config
		backupPath := configPath + ".bak." + time.Now().Format("20060102150405")
		if backupErr := os.Rename(configPath, backupPath); backupErr == nil {
			fmt.Printf("Backed up corrupted config to: %s\n", backupPath)
		}

		// Create new default config
		config = Config{
			Servers: []OllamaServer{
				{Name: "default", Address: "127.0.0.1:11434"},
			},
			Current:    "default",
			ConfigPath: configPath,
		}
		if saveErr := saveConfig(); saveErr != nil {
			fmt.Printf("Error creating new config: %v\n", saveErr)
			os.Exit(1)
		}
		fmt.Printf("Created new default config due to corruption\n")
	}

	// Ensure config has a valid current server
	if config.Current == "" || !serverExists(config.Current) {
		config.Current = "default"
		saveConfig()
	}
}

func saveConfig() error {
	file, err := os.Create(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("error creating config file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("error encoding config file: %v", err)
	}
	return nil
}

// Server management functions
func addServer(name, address string) {
	// Validate name
	if name == "" {
		fmt.Println("Server name cannot be empty")
		return
	}

	// Check address format
	if !strings.Contains(address, ":") {
		address += ":11434" // Default port
	}

	// Check if already exists
	for _, server := range config.Servers {
		if server.Name == name {
			fmt.Printf("Server with name '%s' already exists\n", name)
			return
		}
	}

	// Add new server
	config.Servers = append(config.Servers, OllamaServer{
		Name:    name,
		Address: address,
	})
	if err := saveConfig(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Added server '%s' (%s)\n", name, address)
}

func listServers() {
	fmt.Println("Available Ollama servers:")
	for _, server := range config.Servers {
		currentMark := " "
		if server.Name == config.Current {
			currentMark = "*"
		}
		fmt.Printf("%s %s: %s\n", currentMark, server.Name, server.Address)
	}
}

func useServer(name string) {
	// Find server
	found := false
	for _, server := range config.Servers {
		if server.Name == name {
			config.Current = name
			if err := saveConfig(); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("Now using server '%s' (%s)\n", name, server.Address)
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("Server with name '%s' not found\n", name)
	}
}

func removeServer(name string) {
	if name == "default" {
		fmt.Println("Cannot remove the default server")
		return
	}

	for i, server := range config.Servers {
		if server.Name == name {
			// Remove from slice
			config.Servers = append(config.Servers[:i], config.Servers[i+1:]...)

			// If removing the current server, switch to default
			if config.Current == name {
				config.Current = "default"
			}

			if err := saveConfig(); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("Removed server '%s'\n", name)
			return
		}
	}

	fmt.Printf("Server with name '%s' not found\n", name)
}

func showCurrentServer() {
	for _, server := range config.Servers {
		if server.Name == config.Current {
			fmt.Printf("Current server: %s (%s)\n", server.Name, server.Address)
			return
		}
	}
	fmt.Println("No current server selected")
}

func pingCurrentServer() {
	server := getCurrentServer()
	if server == nil {
		fmt.Println("No current server selected")
		return
	}

	url := fmt.Sprintf("http://%s/api/tags", server.Address)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error pinging server %s: %v\n", server.Address, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Server %s is reachable\n", server.Address)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Server %s returned status: %d (%s)\n", server.Address, resp.StatusCode, string(body))
	}
}

// Helper functions
func serverExists(name string) bool {
	for _, server := range config.Servers {
		if server.Name == name {
			return true
		}
	}
	return false
}

func getCurrentServer() *OllamaServer {
	for i, server := range config.Servers {
		if server.Name == config.Current {
			return &config.Servers[i]
		}
	}
	return nil
}

// Ollama command execution
func runOllamaCommand(command string, args []string) {
	server := getCurrentServer()
	if server == nil {
		fmt.Println("No current server selected")
		return
	}

	// Set OLLAMA_HOST environment variable
	os.Setenv("OLLAMA_HOST", server.Address)

	// Execute ollama command
	cmd := exec.Command("ollama", append([]string{command}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Allow for interactive input

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing ollama command: %v\n", err)
	}
}

// Help functions
func printShortHelp() {
	fmt.Printf(`Ollamanager v%s - %s

Basic Commands:
  ollamanager run <model>              Run a model on current server
  ollamanager server list              List available servers
  ollamanager server use <name>        Switch to a different server

For more detailed help:
  ollamanager help                     Show detailed help
  ollamanager help server              Show server management commands

Current server: %s
`, version, description, getCurrentServerName())
}

func getCurrentServerName() string {
	server := getCurrentServer()
	if server == nil {
		return "none"
	}
	return server.Name
}

func printDetailedHelp() {
	fmt.Printf(`Ollamanager v%s - %s

Usage:
  ollamanager [command]
  ollamanager server [command]

Server Management Commands:
  ollamanager server add <name> <address>    Add a new Ollama server
  ollamanager server list                    List all saved servers
  ollamanager server use <name>              Switch to a specific server
  ollamanager server remove <name>           Remove a server
  ollamanager server current                 Show current server
  ollamanager server ping                    Ping current server to check connectivity

Ollama Commands:
  (all standard ollama commands are supported and will run on the current server)
  ollamanager run <model>                    Run a model
  ollamanager pull <model>                   Pull a model
  ollamanager list                           List models on current server
  ollamanager ps                             List running models on current server
  ollamanager create ...                     Create a model
  ollamanager show ...                       Show model information
  
Special Commands:
  ollamanager version                        Show version information
  ollamanager help                           Show this help information
  ollamanager help server                    Show server management help

Examples:
  ollamanager server add remote1 192.168.1.100       Add server with default port (11434)
  ollamanager server use remote1                     Switch to using remote1
  ollamanager run llama2                             Run llama2 model on current server
  ollamanager list                                   List models on current server
`, version, description)
}

func printServerHelp() {
	fmt.Printf(`Ollamanager - Server Management Commands

Usage:
  ollamanager server [command]

Available Commands:
  add <name> <address>    Add a new Ollama server (address format: host:port or just host)
  list                    List all saved servers
  use <name>              Switch to a specific server
  remove <name>           Remove a server
  current                 Show current server
  ping                    Ping current server to check connectivity

Examples:
  ollamanager server add remote1 192.168.1.100       Add server with default port (11434)
  ollamanager server add remote2 192.168.1.101:8080  Add server with custom port
  ollamanager server use remote1                     Switch to using remote1
`)
}
