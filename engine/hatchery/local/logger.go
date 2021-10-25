package local

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rockbears/log"
)

type Logger interface {
	Logf(fmt string, values ...interface{})
	Errorf(fmt string, values ...interface{})
	Fatalf(fmt string, values ...interface{})
}

func (h *HatcheryLocal) startCmd(name string, cmd *exec.Cmd, logger Logger) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failure due to internal error: unable to capture stdout: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failure due to internal error: unable to capture stderr: %v", err)
	}

	stdoutreader := bufio.NewReader(stdout)
	stderrreader := bufio.NewReader(stderr)

	outchan := make(chan bool)
	go func() {
		for {
			line, err := stdoutreader.ReadString('\n')
			if line != "" {
				logger.Logf(line)
			}
			if err != nil {
				stdout.Close()
				close(outchan)
				return
			}
		}
	}()

	errchan := make(chan bool)
	go func() {
		for {
			line, err := stderrreader.ReadString('\n')
			if line != "" {
				logger.Logf(line)
			}
			if err != nil {
				stderr.Close()
				close(errchan)
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start command: %v", err)
	}

	h.Lock()
	h.workers[name] = workerCmd{cmd: cmd, created: time.Now()}
	h.Unlock()

	<-outchan
	<-errchan
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failure: %v", err)
	}

	return nil
}

type localWorkerLogger struct {
	name string
}

func (l localWorkerLogger) Logf(fmt string, values ...interface{}) {
	fmt = strings.TrimSuffix(fmt, "\n")
	log.Info(context.Background(), "hatchery> local> worker> %s> "+fmt, l.name)
}

func (l localWorkerLogger) Errorf(fmt string, values ...interface{}) {
	fmt = strings.TrimSuffix(fmt, "\n")
	log.Error(context.Background(), "hatchery> local> worker> %s> "+fmt, l.name)
}

func (l localWorkerLogger) Fatalf(fmt string, values ...interface{}) {
	fmt = strings.TrimSuffix(fmt, "\n")
	log.Fatal(context.TODO(), "hatchery> local> worker> %s> "+fmt, l.name)
}
