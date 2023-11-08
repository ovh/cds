package main

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

func CmdResult() *cobra.Command {
	c := &cobra.Command{
		Use:   "result",
		Short: "worker result",
		Long:  `Inside a job, manage run result`,
	}
	c.AddCommand(cmdAddResult())
	return c
}

func cmdAddResult() *cobra.Command {
	c := &cobra.Command{
		Use:     "add",
		Aliases: []string{"new"},
		Short:   "worker result add <type> <result>",
		Long:    `Inside a job, add a run result`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := MustNewWorkerHTTPRequest(http.MethodPost, "/v2/result", nil)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := DoHTTPRequest(ctx, req, nil); err != nil {
				sdk.Exit(err.Error())
			}
			return nil
		},
	}
	return c
}
