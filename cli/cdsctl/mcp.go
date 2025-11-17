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
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

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
		log.Printf("BEGIN %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		switch r.Method {
		case "POST":
			// Delegate completely to MCP SDK
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

		log.Printf("END %s %s", r.Method, r.URL.Path)
	}

	// HTTP mux
	mux := http.NewServeMux()

	// IMPORTANT: Register both /mcp and /mcp/ to avoid implicit ServeMux redirects
	mux.HandleFunc("/mcp", mcpHandler)
	mux.HandleFunc("/mcp/", mcpHandler)

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("MCP HTTP server listening on http://%s (MCP on /mcp, health on /healthz) - %s", addr, modeInfo)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	return nil
}

func registerTools(server *mcp.Server, v cli.Values) {
	logTrace(v, "registerTools", "Registering MCP tools")
	defer logTrace(v, "registerTools", "End Registering MCP tools")

	cmds := FindCommandsByMCPAnnotation(root, v)

	for _, c := range cmds {
		logTrace(v, "registerTools", fmt.Sprintf("Registering MCP tool for command: %s\n", c.Short))

		mcp.AddTool(server, &mcp.Tool{
			Name:        c.Name(),
			Title:       c.Short,
			Description: c.Short + "\n" + c.Example,
		}, func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, map[string]any, error) {
			logTrace(v, "CallTool "+c.Name(), fmt.Sprintf("Inputs: %+v", in))
			var outBuf, errBuf bytes.Buffer
			c.SetOut(&outBuf)
			c.SetErr(&errBuf)

			// Préparer les args (sans le nom de la commande)
			c.SetArgs([]string{"--format", "json"})

			// Parser les flags si besoin explicitement (Execute le fait sinon)
			_ = c.ParseFlags(c.Flags().Args())

			// Si la commande définit RunE
			if c.RunE != nil {
				if err := c.RunE(c, c.Flags().Args()); err != nil {
					// handle error
				}
			} else if c.Run != nil {
				c.Run(c, c.Flags().Args())
			}

			fmt.Printf("OUT: %s\n", outBuf.String())
			fmt.Printf("ERR: %s\n", errBuf.String())

			logTrace(v, "CallTool "+c.Name(), outBuf.String())
			return nil, map[string]any{}, nil
		})
	}
}

func runStdioMode(server *mcp.Server, v cli.Values) {
	logTrace(v, "runStdioMode", "Starting MCP server in stdio mode")

	// Graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// MCP SDK automatically handles stdin/stdout
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logTrace(v, "runStdioMode", fmt.Sprintf("server error: %v", err))
	}

	logTrace(v, "runStdioMode", "server stopped")

}

func FindCommandsByMCPAnnotation(root *cobra.Command, v cli.Values) []*cobra.Command {
	logTrace(v, "FindCommandsByMCPAnnotation", "Start searching cmd")
	defer logTrace(v, "FindCommandsByMCPAnnotation", "End searching cmd")
	var result []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			logTrace(v, "FindCommandsByMCPAnnotation", fmt.Sprintf("Checking command: %s - Ann: %+v\n", sub.Name(), sub.Annotations))
			if sub.Annotations != nil {
				output := sub.Annotations["mcp_output"]
				if output != "" {
					result = append(result, sub)
				}
			}
			walk(sub)
		}
	}
	walk(root)
	logTrace(v, "FindCommandsByMCPAnnotation", fmt.Sprintf("Found %d commands\n", len(result)))
	return result
}

func logTrace(v cli.Values, prefix string, data any) {
	if !v.GetBool("trace") {
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
	if v.GetString("logfile") != "" {
		f, err := os.OpenFile(v.GetString("logfile"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("ERROR opening log file %s: %v", v.GetString("logfile"), err)
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
