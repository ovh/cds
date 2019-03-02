package grpcplugin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// readyString have to be written by plugin, worker read it
const readyString = "is ready to accept new connection\n"

// Plugin is the interface to be implemented by plugin
type Plugin interface {
	Start(context.Context) error
	Stop(context.Context, *empty.Empty) (*empty.Empty, error)
	Instance() *Common
}

// StartPlugin starts a plugin, returns stdoutPipe, stderrPipe and socketName
func StartPlugin(ctx context.Context, pluginName string, workdir, cmd string, args []string, env []string) (io.Reader, string, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = workdir
	c.Env = env
	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return nil, "", err
	}
	stderrPipe, err := c.StderrPipe()
	if err != nil {
		return nil, "", err
	}

	r1 := bufio.NewReader(stdoutPipe)
	r2 := bufio.NewReader(stderrPipe)
	reader := io.MultiReader(r1, r2)

	if err := c.Start(); err != nil {
		return nil, "", err
	}

	go func() {
		if err := c.Wait(); err != nil {
			log.Info("GRPC Plugin %s wait failed:%+v", cmd, err)
		}
		log.Info("GRPC Plugin %s end", cmd)
	}()

	log.Info("GRPC Plugin %s started", cmd)

	//Sleep a while, to let the plugin write on stdout the socket address
	time.Sleep(500 * time.Millisecond)
	tsStart := time.Now()

	stdoutreader := bufio.NewReader(stdoutPipe)

	var socket string
	var errReturn error
	for {
		line, errs := stdoutreader.ReadString('\n')
		if errs == io.EOF {
			continue
		}
		if errs != nil {
			if time.Now().Before(tsStart.Add(5 * time.Second)) {
				log.Warning("plugin:%s error on ReadString, retry in 500ms...", pluginName)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			errReturn = fmt.Errorf("plugin:%s error on ReadString(len buff %d, content: %s): %v", pluginName, len(line), line, err)
			break
		}
		if strings.HasSuffix(line, readyString) {
			socket = strings.TrimSpace(strings.Replace(line, fmt.Sprintf(" %s", readyString), "", 1))
			log.Info("socket %s ready", socket)
			break
		}
	}
	return reader, socket, errReturn
}

type Common struct {
	Desc   *grpc.ServiceDesc
	Srv    interface{}
	Socket string
	s      *grpc.Server
}

func (c *Common) Instance() *Common {
	return c
}

func (c *Common) Start(ctx context.Context) error {
	_, err := c.start(ctx, c.Desc, c.Srv)
	return err
}

func userCacheDir() string {
	cdir := os.Getenv("HOME_CDS_PLUGINS")
	if cdir == "" {
		cdir = os.TempDir()
	}

	switch runtime.GOOS {
	case "windows":
		cdir = os.Getenv("LocalAppData")
	case "darwin":
		cdir += "/Library/Caches"
	case "plan9":
		cdir += "/lib/cache"
	default: // Unix
		dir := os.Getenv("XDG_CACHE_HOME")
		if dir != "" {
			cdir = dir
		}
	}

	return cdir
}

func (c *Common) start(ctx context.Context, desc *grpc.ServiceDesc, srv interface{}) (Plugin, error) {
	//Start the grpc server on unix socket
	uuid := sdk.UUID()
	c.Socket = filepath.Join(userCacheDir(), fmt.Sprintf("grpcplugin-socket-%s.sock", uuid))
	syscall.Unlink(c.Socket)
	l, err := net.Listen("unix", c.Socket)
	if err != nil {
		return nil, fmt.Errorf("unable to listen on socket %s: %v", c.Socket, err)
	}

	s := grpc.NewServer()
	c.s = s
	c.s.RegisterService(desc, srv)
	reflection.Register(s)

	go func() {
		<-ctx.Done()
		fmt.Printf("exiting plugin\n")
		defer os.RemoveAll(c.Socket)
		c.s.Stop()
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		socket, _ := filepath.Abs(c.Socket)
		fmt.Printf("%s %s", socket, readyString)
	}()

	return c, s.Serve(l)
}

func (c *Common) Stop(context.Context, *empty.Empty) (*empty.Empty, error) {
	defer func() {
		fmt.Printf("Stopping plugin...")
		time.Sleep(2 * time.Second)
		c.s.Stop()
	}()
	return new(empty.Empty), nil
}

// InfoMarkdown returns string formatted with markdown
func InfoMarkdown(pl sdk.GRPCPlugin) string {
	var sp string
	sort.Slice(pl.Parameters, func(i, j int) bool {
		return pl.Parameters[i].Name < pl.Parameters[j].Name
	})

	for _, param := range pl.Parameters {
		sp += fmt.Sprintf("* **%s**: %s\n", param.Name, param.Description)
	}

	info := fmt.Sprintf(`
%s

## Parameters

%s

`,
		pl.Description,
		sp)

	return info
}
