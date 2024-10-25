package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdExit() *cobra.Command {
	c := &cobra.Command{
		Use:   "exit",
		Short: "worker exit",
		Long:  "worker exit command lets job finish current step with exit code 0 (success) and disabled all further steps",
		Run:   exitCmd(),
	}
	return c
}

func exitCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		port := MustGetWorkerHTTPPort()

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/exit", port), nil)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		if errRequest != nil {
			sdk.Exit("cannot post worker exit (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 5 * time.Second

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("command failed: %v\n", errDo)
		}

		if resp.StatusCode >= 300 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("tag failed: unable to read body %v\n", err)
			}
			defer resp.Body.Close()
			sdk.Exit("exit failed: %s\n", string(body))
		}
	}
}
