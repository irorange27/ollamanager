# Ollamanager

> [!NOTE]
This CLI tool is built by Github Copilot and Claude 3.7 Sinnet Thinking

A wrapper for ollama that allows controlling ollama instances available on your internal network.

## Description

Ollamanager makes it easy to manage multiple ollama servers and switch between them. It acts as a command wrapper that forwards all ollama commands to the currently selected server, allowing you to work with remote ollama instances as if they were running locally.

## Installation

### Using Go Install

```bash
go install github.com/irorange27/ollamanager@latest
```

### From Source

1. Clone the repository:

   ```bash
   git clone https://github.com/irorange27/ollamanager.git
   cd ollamanager
   ```

2. Build the binary:

   ```bash
   go build -o ollamanager ./cmd/ollamanager
   ```

3. Move it to a directory in your PATH:

   ```bash
   sudo mv ollamanager /usr/local/bin/
   ```

## Usage

```bash
# Server management
ollamanager server add <name> <address>      # Add a new server
ollamanager server list                      # List all servers
ollamanager server use <name>                # Switch to a server
ollamanager server remove <name>             # Remove a server
ollamanager server ping                      # Check if current server is reachable

# Run ollama commands on the current server
ollamanager run llama2                       # Run the llama2 model on current server
ollamanager pull mistral                     # Pull a model from the current server
ollamanager list                             # List models on the current server
```

All standard ollama commands are supported and will be forwarded to the currently selected server.

## Examples

```bash
# Add a local server
ollamanager server add local 127.0.0.1

# Add a remote server
ollamanager server add workstation 192.168.1.100

# Switch to the remote server
ollamanager server use workstation

# Run a model on the remote server
ollamanager run mistral
```

## Disclaimer

This software is provided for legitimate use only. Users are responsible for ensuring their usage complies with all applicable laws and regulations. The authors do not condone or support illegal activities of any kind. Use this tool responsibly and ethically.

## License

MIT
