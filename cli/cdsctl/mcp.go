package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

var mcpLog *mcpLogger

var mcpCmd = cli.Command{
	Name:    "mcp",
	Aliases: []string{""},
	Short:   "Start mcp",
}

func mcpCommands() *cobra.Command {
	return cli.NewCommand(mcpCmd, nil, []*cobra.Command{
		cli.NewCommand(mcpStartCmd, mcpStartRun, nil, withAllCommandModifiers()...),
	})
}

var mcpStartCmd = cli.Command{
	Name:  "start",
	Short: "Start mcp server",
	Ctx:   []cli.Arg{},
	Flags: []cli.Flag{
		{
			Name: "mode",
			IsValid: func(v string) bool {
				switch v {
				case "stdio", "sse", "streamable_http":
					return true
				default:
					fmt.Println("invalid mode, allowed values are: stdio, sse, streamable_http")
					return false
				}
			},
		},
		{
			Name: "trace",
			Type: cli.FlagBool,
		},
		{
			Name:    "logfile",
			Default: "/tmp/mcp_cds.log",
		},
	},
}

func mcpStartRun(v cli.Values) error {
	mcpLog = &mcpLogger{
		traceEnabled: v.GetBool("trace"),
		logFile:      v.GetString("logfile"),
	}

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "cds-mcp",
			Version: sdk.VERSION,
		},
		&mcp.ServerOptions{
			Instructions: ``,
			HasTools:     true,
			KeepAlive:    30 * time.Second,
		},
	)
	registerTools(server, v)

	if v.GetString("mode") == "stdio" {
		runStdioMode(server, v)
		return nil
	}

	// Network binding for HTTP modes
	host := os.Getenv("MCP_HTTP_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("MCP_HTTP_PORT")
	if port == "" {
		port = "8000"
	}
	addr := host + ":" + port

	// Main MCP handler
	mcpHandler := func(w http.ResponseWriter, r *http.Request) {
		mcpLog.logTrace("Hander"+r.Method+" "+r.URL.Path, "BEGIN")
		defer mcpLog.logTrace("Hander"+r.Method+" "+r.URL.Path, "END")
		switch r.Method {
		case "POST":
			if v.GetString("mode") == "sse" {
				mcp.NewSSEHandler(
					func(r *http.Request) *mcp.Server { return server },
					&mcp.SSEOptions{},
				).ServeHTTP(w, r)
			} else { // streamable_http
				mcp.NewStreamableHTTPHandler(
					func(r *http.Request) *mcp.Server { return server },
					&mcp.StreamableHTTPOptions{
						Stateless:    true,
						JSONResponse: true,
					},
				).ServeHTTP(w, r)
			}
		default:
			// Don't return fake JSON-RPC, goose hates it
			w.Header().Set("Allow", "POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}

	// HTTP mux
	mux := http.NewServeMux()
	// IMPORTANT: Register both /mcp and /mcp/ to avoid implicit ServeMux redirects
	mux.HandleFunc("/mcp", mcpHandler)
	mux.HandleFunc("/mcp/", mcpHandler)
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	// HTTP server
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start HTTP server asynchronously
	go func() {
		modeInfo := fmt.Sprintf("mode=%s", v.GetString("mode"))
		log.Printf("MCP HTTP server listening on http://%s (MCP on /mcp, health on /health) - %s\n", addr, modeInfo)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	return nil
}

type mcpLogger struct {
	traceEnabled bool
	logFile      string
}

func (ml *mcpLogger) logTrace(prefix string, data any) {
	if !ml.traceEnabled {
		return
	}

	var message string
	switch v := data.(type) {
	case string:
		const max = 4096
		if len(v) > max {
			v = v[:max] + "...(truncated)"
		}
		message = fmt.Sprintf("TRACE %s: %s", prefix, v)
	default:
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			message = fmt.Sprintf("TRACE %s: [ERROR marshaling data: %v]", prefix, err)
		} else {
			message = fmt.Sprintf("TRACE %s: %s", prefix, string(jsonData))
		}
	}

	// Log to file if specified, otherwise to standard log
	if ml.logFile != "" {
		f, err := os.OpenFile(ml.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("ERROR opening log file %s: %v", ml.logFile, err)
			log.Printf("%s", message) // fallback to standard log
			return
		}
		defer f.Close()

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(f, "[%s] %s\n", timestamp, message)
	} else {
		log.Printf("%s", message)
	}
}

func registerTools(server *mcp.Server, v cli.Values) {
	mcpLog.logTrace("registerTools", "Registering MCP tools")
	defer mcpLog.logTrace("registerTools", "End Registering MCP tools")

	cmds := FindCommandsByMCPAnnotation(root, v)

	for _, c := range cmds {
		mcpLog.logTrace("registerTools", fmt.Sprintf("Registering MCP tool for command: %s\n", c.Name))

		cmd := c
		mcp.AddTool(server, &mcp.Tool{
			Name:        cmd.Name,
			Title:       cmd.Cmd.Short,
			Description: cmd.Cmd.Short + "\n" + cmd.Cmd.Example,
			InputSchema: cmd.Inputs,
		}, func(ctx context.Context, req *mcp.CallToolRequest, in map[string]string) (*mcp.CallToolResult, map[string]any, error) {
			mcpLog.logTrace("CallTool "+cmd.Name, fmt.Sprintf("Inputs: %+v", in))

			var outBuf, errBuf bytes.Buffer
			cmd.Cmd.SetOut(&outBuf)
			cmd.Cmd.SetErr(&errBuf)
			args := make([]string, 0)
			for _, arg := range cmd.Args {
				i, found := in[arg.Name]
				if !found {
					mcpLog.logTrace("CallTool "+cmd.Name+" error:", "missing argument: "+arg.Name)
					return nil, nil, fmt.Errorf("missing argument: %s", arg.Name)
				}
				args = append(args, i)
			}

			mcpLog.logTrace("CallTool "+cmd.Name, fmt.Sprintf("Args: %+v", args))

			// Parse flag before execution
			if err := cmd.Cmd.ParseFlags([]string{"--format", "json"}); err != nil {
				mcpLog.logTrace("CallTool "+cmd.Name+" parse error:", err.Error())
				return nil, nil, err
			}

			cmd.Cmd.Run(cmd.Cmd, args)

			mcpLog.logTrace("CallTool "+cmd.Name+" out:", outBuf.String())
			mcpLog.logTrace("CallTool "+cmd.Name+" err:", errBuf.String())

			if outBuf.Len() == 0 {
				if errBuf.Len() > 0 {
					mcpLog.logTrace("CallTool "+cmd.Name+" error:", errBuf.String())
					return nil, nil, fmt.Errorf("command failed: %s", errBuf.String())
				}
				mcpLog.logTrace("CallTool "+cmd.Name+" error:", "command produced no output")
				return nil, nil, fmt.Errorf("command produced no output")
			}

			outpoutString := outBuf.String()
			if strings.HasPrefix(outpoutString, "[") {
				var out []any
				if err := json.Unmarshal(outBuf.Bytes(), &out); err != nil {
					return nil, nil, err
				}
				return nil, map[string]any{"values": out}, nil
			} else {
				var out map[string]any
				if err := json.Unmarshal(outBuf.Bytes(), &out); err != nil {
					return nil, nil, err
				}
				return nil, out, nil
			}

		})
	}
}

func runStdioMode(server *mcp.Server, v cli.Values) {
	mcpLog.logTrace("runStdioMode", "Starting MCP server in stdio mode")

	// Graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// MCP SDK automatically handles stdin/stdout
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		mcpLog.logTrace("runStdioMode", fmt.Sprintf("server error: %v", err))
	}

	mcpLog.logTrace("runStdioMode", "server stopped")

}

type MCPCommand struct {
	Name   string
	Cmd    *cobra.Command
	Args   []cli.Arg
	Inputs *jsonschema.Schema
}

func FindCommandsByMCPAnnotation(root *cobra.Command, v cli.Values) []MCPCommand {
	mcpLog.logTrace("FindCommandsByMCPAnnotation", "Start searching cmd")
	defer mcpLog.logTrace("FindCommandsByMCPAnnotation", "End searching cmd")
	var result []MCPCommand
	var walk func(c *cobra.Command, parentCommandPrefix string)
	walk = func(c *cobra.Command, prefix string) {
		for _, sub := range c.Commands() {
			if sub.Annotations != nil {
				output := sub.Annotations["mcp"]
				if output != "" {
					mcpLog.logTrace("FindCommandsByMCPAnnotation", fmt.Sprintf("Adding command: %s-%s\n", prefix, sub.Name()))
					mcpCommand := MCPCommand{
						Name: prefix + "-" + sub.Name(),
						Cmd:  sub,
						Inputs: &jsonschema.Schema{
							Type:       "object",
							Properties: map[string]*jsonschema.Schema{},
						},
					}
					_ = json.Unmarshal([]byte(output), &mcpCommand.Args)
					for _, arg := range mcpCommand.Args {
						mcpCommand.Inputs.Properties[arg.Name] = &jsonschema.Schema{Type: "string"}
					}
					result = append(result, mcpCommand)
				}
			}
			if prefix == "" {
				walk(sub, sub.Name())
			} else {
				walk(sub, prefix+"-"+sub.Name())
			}
		}
	}
	walk(root, "")
	mcpLog.logTrace("FindCommandsByMCPAnnotation", fmt.Sprintf("Found %d commands\n", len(result)))
	return result
}
