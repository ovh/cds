package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func generateDocumentation(root *cobra.Command) error {
	const fmTemplate = `+++
title = "%s"
+++
`

	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		base = strings.Replace(base, "cdsctl_", "", 1)
		return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		base = strings.Replace(base, "cdsctl_", "", 1)
		return "/cli/commands/" + strings.Replace(strings.ToLower(base), "_", "/", -1) + "/"
	}

	return genMarkdownTreeCustom(root, "./commands", filePrepender, linkHandler)
	// 	return err
	// }

	// f, err := os.Create("./commands/_index.md")
	// if err != nil {
	// 	return err
	// }
	// defer f.Close()
	// fmt.Println("create file ./commands/_index.md")
	// if _, err := io.WriteString(f, filePrepender("cdsctl")); err != nil {
	// 	return err
	// }
	// root.DisableAutoGenTag = true
	// return doc.GenMarkdownCustom(root, f, linkHandler)
}

// genMarkdownTreeCustom is the the same as GenMarkdownTree, but
// with custom filePrepender and linkHandler.
// this func is inspired from https://github.com/spf13/cobra/blob/master/doc/md_docs.go
func genMarkdownTreeCustom(cmd *cobra.Command, rootdir string, filePrepender, linkHandler func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := genMarkdownTreeCustom(c, rootdir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	var basename, basenameTitle string
	a := strings.Split(cmd.CommandPath(), " ")
	withoutBinary := a[1:] // remove 'cdsctl'
	if len(withoutBinary) > 0 {
		basename = strings.Join(withoutBinary, "/")

		// create directory, only for command containing commands
		if len(withoutBinary) > len(a)-2 {
			dir := strings.Join(withoutBinary[:len(a)-2], "/")
			if dir != "" {
				fmt.Printf("create directory %s/%s\n", rootdir, dir)
				os.MkdirAll(rootdir+"/"+dir, os.ModePerm)
			}
		}

		basenameTitle = basename + ".md"
		// if the command name already exists as a directory,
		// we have to create a _index.md file
		if _, err := os.Stat(rootdir + "/" + basename); err == nil {
			basename += "/_index"
		}
	} else { // root cmd
		basenameTitle = "cdsctl commands"
		basename += "_index"
	}
	basename += ".md"
	fmt.Printf("create file %s/%s\n", rootdir, basename)
	filename := filepath.Join(rootdir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(basenameTitle)); err != nil {
		return err
	}
	cmd.DisableAutoGenTag = true
	return doc.GenMarkdownCustom(cmd, f, linkHandler)
}
