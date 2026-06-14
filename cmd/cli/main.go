package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "login":
		login(args)
	case "tsnet":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: homelytics-agent tsnet auth")
			os.Exit(1)
		}
		switch args[0] {
		case "auth":
			tsnetAuth(args[1:])
		default:
			fmt.Fprintf(os.Stderr, "unknown tsnet subcommand: %s\n", args[0])
			os.Exit(1)
		}
	case "runtime":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: homelytics-agent runtime status")
			os.Exit(1)
		}
		switch args[0] {
		case "status":
			runtimeStatus(args[1:])
		default:
			fmt.Fprintf(os.Stderr, "unknown runtime subcommand: %s\n", args[0])
			os.Exit(1)
		}
	case "status":
		status(args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: homelytics-agent <command> [options]

Commands:
  login --email=<email> --password=<password>
  tsnet auth
  runtime status
  status

Global flags (for client commands):
  --socket-path=<path>  Override IPC socket path (default from config).
  --config=<path>       Config file path (default files/config/app.yaml).`)
}

func login(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "Merchant email")
	password := fs.String("password", "", "Merchant password")
	cli := parseClientFlags(fs, args)

	payload, _ := json.Marshal(entity.LoginRequest{Email: *email, Password: *password})
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "login", Payload: payload})
	if err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func tsnetAuth(args []string) {
	fs := flag.NewFlagSet("tsnet auth", flag.ExitOnError)
	cli := parseClientFlags(fs, args)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "tsnet.auth"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "tsnet auth failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func runtimeStatus(args []string) {
	fs := flag.NewFlagSet("runtime status", flag.ExitOnError)
	cli := parseClientFlags(fs, args)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "runtime.status"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "runtime status failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func status(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	cli := parseClientFlags(fs, args)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "status"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

type ipcClient struct {
	socketPath string
}

func parseClientFlags(fs *flag.FlagSet, args []string) *ipcClient {
	configPath := fs.String("config", "files/config/app.yaml", "Path to config file")
	socketOverride := fs.String("socket-path", "", "Override IPC socket path")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "flag parse error: %v\n", err)
		os.Exit(1)
	}

	socketPath := *socketOverride
	if socketPath == "" {
		cfg, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}
		socketPath = cfg.IPC.SocketPath
	}

	return &ipcClient{socketPath: socketPath}
}

func (c *ipcClient) send(ctx context.Context, req entity.CommandRequest) (*entity.CommandResponse, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", c.socketPath, err)
	}
	defer conn.Close()

	data, _ := json.Marshal(req)
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	if _, err := conn.Write([]byte("\n")); err != nil {
		return nil, fmt.Errorf("send newline: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("empty response")
	}

	var resp entity.CommandResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &resp, nil
}

func printResponse(resp *entity.CommandResponse) {
	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(data))
}

func ctx() context.Context {
	return context.Background()
}
