package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// globalOpts holds flags that may appear before or after the subcommand.
type globalOpts struct {
	configPath     string
	socketOverride string
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	globals, cmd, args := parseGlobalFlags(os.Args[1:])
	if cmd == "" {
		printUsage()
		os.Exit(1)
	}

	switch cmd {
	case "login":
		login(globals, args)
	case "tsnet":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: homelytics-agent tsnet auth")
			os.Exit(1)
		}
		switch args[0] {
		case "auth":
			tsnetAuth(globals, args[1:])
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
			runtimeStatus(globals, args[1:])
		default:
			fmt.Fprintf(os.Stderr, "unknown runtime subcommand: %s\n", args[0])
			os.Exit(1)
		}
	case "workload":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: homelytics-agent workload [run|list|status|stop|delete]")
			os.Exit(1)
		}
		switch args[0] {
		case "run":
			workloadRun(globals, args[1:])
		case "list":
			workloadList(globals, args[1:])
		case "status":
			workloadStatus(globals, args[1:])
		case "stop":
			workloadStop(globals, args[1:])
		case "delete":
			workloadDelete(globals, args[1:])
		default:
			fmt.Fprintf(os.Stderr, "unknown workload subcommand: %s\n", args[0])
			os.Exit(1)
		}
	case "status":
		status(globals, args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

// parseGlobalFlags extracts global flags that may appear before the subcommand.
// It returns the parsed globals, the subcommand name, and the remaining args.
func parseGlobalFlags(args []string) (globalOpts, string, []string) {
	var g globalOpts
	fs := flag.NewFlagSet("global", flag.ContinueOnError)
	fs.StringVar(&g.configPath, "config", "files/config/app.yaml", "Path to config file")
	fs.StringVar(&g.socketOverride, "socket-path", "", "Override IPC socket path")
	fs.Usage = printUsage

	// Parse only global flags; stop at the first non-flag argument (the subcommand).
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "flag parse error: %v\n", err)
		os.Exit(1)
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return g, "", nil
	}

	return g, remaining[0], remaining[1:]
}

func printUsage() {
	fmt.Println(`Usage: homelytics-agent [global flags] <command> [command flags]

Global flags:
  --config=<path>       Config file path (default files/config/app.yaml)
  --socket-path=<path>  Override IPC socket path (default from config)

Commands:
  login --email=<email> --password=<password>
  tsnet auth
  runtime status
  workload run --image=<image> [--id=<id>] [--port=<host>:<container>] [--host-network]
  workload list
  workload status --id=<id>
  workload stop --id=<id>
  workload delete --id=<id>
  status

Examples:
  homelytics-agent --config /opt/homelytics/etc/config.yaml login --email merchant@example.com --password password
  homelytics-agent --socket-path ./var/run/ipc.sock status
  homelytics-agent workload run --image nginx:latest --port 8080:80 --host-network`)
}

func login(g globalOpts, args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "Merchant email")
	password := fs.String("password", "", "Merchant password")
	cli := buildClient(g, fs, args)

	payload, _ := json.Marshal(entity.LoginRequest{Email: *email, Password: *password})
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "login", Payload: payload})
	if err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func tsnetAuth(g globalOpts, args []string) {
	fs := flag.NewFlagSet("tsnet auth", flag.ExitOnError)
	cli := buildClient(g, fs, args)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "tsnet.auth"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "tsnet auth failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func runtimeStatus(g globalOpts, args []string) {
	fs := flag.NewFlagSet("runtime status", flag.ExitOnError)
	cli := buildClient(g, fs, args)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "runtime.status"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "runtime status failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func status(g globalOpts, args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	cli := buildClient(g, fs, args)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "status"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func workloadRun(g globalOpts, args []string) {
	fs := flag.NewFlagSet("workload run", flag.ExitOnError)
	image := fs.String("image", "", "Container image reference")
	id := fs.String("id", "", "Workload ID (generated if empty)")
	hostNetwork := fs.Bool("host-network", true, "Run in host network namespace")
	var ports portFlag
	fs.Var(&ports, "port", "Port mapping host:container (repeatable)")
	cli := buildClient(g, fs, args)

	if *image == "" {
		fmt.Fprintln(os.Stderr, "workload run requires --image")
		os.Exit(1)
	}

	req := entity.RunWorkloadRequest{
		ID:          *id,
		Image:       *image,
		Ports:       ports.mappings,
		HostNetwork: *hostNetwork,
	}
	payload, _ := json.Marshal(req)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "workload.run", Payload: payload})
	if err != nil {
		fmt.Fprintf(os.Stderr, "workload run failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func workloadList(g globalOpts, args []string) {
	fs := flag.NewFlagSet("workload list", flag.ExitOnError)
	cli := buildClient(g, fs, args)
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "workload.list"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "workload list failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func workloadStatus(g globalOpts, args []string) {
	fs := flag.NewFlagSet("workload status", flag.ExitOnError)
	id := fs.String("id", "", "Workload ID")
	cli := buildClient(g, fs, args)

	payload, _ := json.Marshal(entity.WorkloadIDRequest{ID: *id})
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "workload.status", Payload: payload})
	if err != nil {
		fmt.Fprintf(os.Stderr, "workload status failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func workloadStop(g globalOpts, args []string) {
	fs := flag.NewFlagSet("workload stop", flag.ExitOnError)
	id := fs.String("id", "", "Workload ID")
	cli := buildClient(g, fs, args)

	payload, _ := json.Marshal(entity.WorkloadIDRequest{ID: *id})
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "workload.stop", Payload: payload})
	if err != nil {
		fmt.Fprintf(os.Stderr, "workload stop failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

func workloadDelete(g globalOpts, args []string) {
	fs := flag.NewFlagSet("workload delete", flag.ExitOnError)
	id := fs.String("id", "", "Workload ID")
	cli := buildClient(g, fs, args)

	payload, _ := json.Marshal(entity.WorkloadIDRequest{ID: *id})
	resp, err := cli.send(ctx(), entity.CommandRequest{Method: "workload.delete", Payload: payload})
	if err != nil {
		fmt.Fprintf(os.Stderr, "workload delete failed: %v\n", err)
		os.Exit(1)
	}
	printResponse(resp)
}

type portFlag struct {
	mappings map[string]string
}

func (f *portFlag) String() string {
	return ""
}

func (f *portFlag) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("port mapping must be host:container, got %q", value)
	}
	if f.mappings == nil {
		f.mappings = make(map[string]string)
	}
	f.mappings[parts[1]] = parts[0]
	return nil
}

type ipcClient struct {
	socketPath string
}

// buildClient parses command-specific flags and resolves the IPC socket path.
func buildClient(g globalOpts, fs *flag.FlagSet, args []string) *ipcClient {
	// Re-add the global flags to each subcommand flag set so they can appear
	// after the subcommand as well (e.g. "login --config ... --email ...").
	configPath := fs.String("config", g.configPath, "Path to config file")
	socketOverride := fs.String("socket-path", g.socketOverride, "Override IPC socket path")

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
