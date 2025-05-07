package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
)

func CmdOutput() *cobra.Command {
	c := &cobra.Command{
		Use:     "output",
		Aliases: []string{"export"},
		Short:   "worker output <output_name> <output_value>",
		Long:    `Inside a job, create an output available through the jobs and steps contexts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var value string
			if len(args) == 1 {
				stdin, err := io.ReadAll(os.Stdin)
				if err != nil {
					sdk.Exit("Error reading stdin: %v", err)
				}
				value = string(stdin)
			} else if len(args) == 2 {
				value = args[1]
			} else {
				sdk.Exit("wrong number of arguments. Need 2, Got [%d]", len(args))
			}

			outputRequest := workerruntime.OutputRequest{
				Name:  args[0],
				Value: value,
			}
			req := MustNewWorkerHTTPRequest(http.MethodPost, "/v2/output", outputRequest)
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
