package main

import (
	"fmt"
	"log"
	"os"

	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Value:   "config.yml",
			EnvVars: []string{"SMTPMOCK_CONFIG"},
		},
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "smtp-port",
			Value:   2023,
			EnvVars: []string{"SMTPMOCK_SMTP_PORT"},
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "api-port",
			Value:   2024,
			EnvVars: []string{"SMTPMOCK_API_PORT"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:    "with-auth",
			EnvVars: []string{"SMTPMOCK_SMTP_WITH_AUTH"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "jwt-secret",
			EnvVars: []string{"SMTPMOCK_SMTP_JWT_SECRET"},
		}),
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "start",
				Usage:  "Starts smtp mock server",
				Action: start,
			},
			{
				Name:   "generate-token",
				Usage:  "Generates a new token for smtp mock client",
				Action: generateToken,
			},
		},
		Before: altsrc.InitInputSourceWithContext(flags,
			func(c *cli.Context) (altsrc.InputSourceContext, error) {
				i, err := altsrc.NewYamlSourceFromFlagFunc("config")(c)
				if err == nil {
					return i, nil
				}
				return &altsrc.MapInputSource{}, nil
			},
		),
		Flags: flags,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("%+v\n", err)
	}
}

func start(ctx *cli.Context) error {
	go func() {
		if err := StartSMTP(ctx.Context, ctx.Int("smtp-port")); err != nil {
			log.Fatal(err)
		}
	}()

	return StartAPI(ctx.Context, ConfigAPI{
		Port:      ctx.Int("api-port"),
		PortSMTP:  ctx.Int("smtp-port"),
		WithAuth:  ctx.Bool("with-auth"),
		JwtSecret: ctx.String("jwt-secret"),
	})
}

func generateToken(ctx *cli.Context) error {
	if !ctx.Bool("with-auth") {
		fmt.Println("Auth not active for given config")
		return nil
	}

	if err := InitJWT([]byte(ctx.String("jwt-secret"))); err != nil {
		return err
	}

	subjectID, token, err := NewSigninToken()
	if err != nil {
		return err
	}

	fmt.Printf("New signin token (%s):\n%s\n", subjectID, token)
	return nil
}
