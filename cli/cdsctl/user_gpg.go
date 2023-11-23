package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var userGpgCmd = cli.Command{
	Name:    "gpg",
	Aliases: []string{"gpg"},
	Short:   "Manage CDS user gpg keys",
}

func userGpg() *cobra.Command {
	return cli.NewCommand(userGpgCmd, nil, []*cobra.Command{
		cli.NewCommand(userGpgKeyShowCmd, userGpgKeyShow, nil),
		cli.NewListCommand(userGpgKeyListCmd, userGpgKeyList, nil),
		cli.NewDeleteCommand(userGpgKeyDeleteCmd, userGpgKeyDelete, nil),
		cli.NewCommand(userGpgKeyImportCmd, userGpgKeyImport, nil),
	})
}

var userGpgKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS users gpg keys",
}

func userGpgKeyList(v cli.Values) (cli.ListResult, error) {
	u, err := client.UserGetMe(context.Background())
	if err != nil {
		return nil, err
	}
	keys, err := client.UserGpgKeyList(context.Background(), u.Username)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}

var userGpgKeyShowCmd = cli.Command{
	Name:  "show",
	Short: "Show Current CDS user gpg key",
	Args: []cli.Arg{
		{
			Name: "keyId",
		},
	},
}

func userGpgKeyShow(v cli.Values) error {
	k, err := client.UserGpgKeyGet(context.Background(), v.GetString("keyId"))
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", k.PublicKey)
	return nil
}

var userGpgKeyDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete CDS user gpg key",
	Args: []cli.Arg{
		{
			Name: "keyId",
		},
	},
}

func userGpgKeyDelete(v cli.Values) error {
	u, err := client.UserGetMe(context.Background())
	if err != nil {
		return err
	}

	if err := client.UserGpgKeyDelete(context.Background(), u.Username, v.GetString("keyId")); err != nil {
		return err
	}
	return nil
}

var userGpgKeyImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a CDS user gpg key",
	Flags: []cli.Flag{
		{
			Name:      "pub-key-file",
			ShortHand: "k",
		},
	},
}

func userGpgKeyImport(v cli.Values) error {
	var publicKey string
	if v.GetString("pub-key-file") == "" {
		// read from stdin
		fmt.Printf("Copy your public key here: \n")

		keyBuilder := strings.Builder{}
		for {
			keyPart := cli.ReadLine()
			keyBuilder.WriteString(keyPart)

			if strings.Contains(keyPart, "END PGP") {
				break

			}
			keyBuilder.WriteString("\n")
		}
		publicKey = keyBuilder.String()
	} else {
		keyBts, err := os.ReadFile(v.GetString("pub-key-file"))
		if err != nil {
			return err
		}
		publicKey = string(keyBts)
	}

	u, err := client.UserGetMe(context.Background())
	if err != nil {
		return err
	}

	key, err := client.UserGpgKeyCreate(context.Background(), u.Username, publicKey)
	if err != nil {
		return err
	}
	fmt.Printf("Gpg key %s created.\n", key.KeyID)
	return nil
}
