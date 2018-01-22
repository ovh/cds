package doc

import (
	"fmt"
	"go/ast"
	godoc "go/doc"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/ovh/cds/sdk"
)

// GenerateDocumentation generates hugo documentation for a command line
func GenerateDocumentation(root *cobra.Command, genPath, gitPath string) error {
	const fmTemplate = `+++
title = "%s"
+++
`
	rootName := root.Name()
	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		base = strings.Replace(base, rootName+"_", "", 1)
		return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		base = strings.Replace(base, rootName+"_", "", 1)
		return fmt.Sprintf("/cli/%s/%s/", rootName, strings.Replace(strings.ToLower(base), "_", "/", -1))
	}

	fmt.Printf("%s\n", rootName)
	if err := os.MkdirAll(genPath+"/"+rootName, os.ModePerm); err != nil {
		return err
	}

	if gitPath != "" {
		if err := os.MkdirAll(genPath+"/api", os.ModePerm); err != nil {
			return err
		}

		/* Example doc on Handler:

		// getActionsHandler Retrieve all public actions
		// @title A title
		// @description the description
		// @params AA=valA
		// @params BB=valB
		// @body {"mykey": "myval"}
		func (api *API) getActionsHandler() Handler {
		[...]
		*/
		if err := writeRouteInfo(getAllRouteInfo(gitPath+"/engine/api"), genPath+"/api"); err != nil {
			return err
		}
	}

	return genMarkdownTreeCustom(root, genPath+"/"+rootName, filePrepender, linkHandler)
}

// genMarkdownTreeCustom is the the same as GenMarkdownTree, but
// with custom filePrepender and linkHandler.
// this func is inspired from https://github.com/spf13/cobra/blob/master/doc/md_docs.go
func genMarkdownTreeCustom(cmd *cobra.Command, rootdir string, filePrepender, linkHandler func(string) string) error {
	cmdName := cmd.Name()
	if cmd.Hidden {
		return nil
	}
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
	withoutBinary := a[1:] // remove cmdName. ex:'cdsctl'
	if len(withoutBinary) > 0 {
		basename = strings.Join(withoutBinary, "/")

		// create directory, only for command containing commands
		if len(withoutBinary) > len(a)-2 {
			dir := strings.Join(withoutBinary[:len(a)-2], "/")
			if dir != "" {
				fmt.Printf("create directory %s/%s\n", rootdir, dir)
				if err := os.MkdirAll(rootdir+"/"+dir, os.ModePerm); err != nil {
					return err
				}
			}
		}

		basenameTitle = basename + ".md"
		// if the command name already exists as a directory,
		// we have to create a _index.md file
		if _, err := os.Stat(rootdir + "/" + basename); err == nil {
			basename += "/_index"
		}
	} else { // root cmd
		basenameTitle = cmdName
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

const (
	title       = "@title"
	description = "@description"
	queryParam  = "@params"
	body        = "@body"
)

// Method Represent data on a method
type Method struct {
	Doc  string
	Name string
}

// Doc represents elements wanted in the documentation
type Doc struct {
	Title         string
	Description   string
	Method        string
	URL           string
	QueryParams   []string
	Body          string
	Middleware    []Middleware
	HTTPOperation string
}

// RouteInfo Information on a route
type RouteInfo struct {
	URL           string
	URLParams     []string
	Method        string
	Middleware    []Middleware
	HTTPOperation string
}

// Middleware represents a middleware
type Middleware struct {
	Name  string
	Value []string
}

type visitor struct {
	processingNewRoute bool

	allRoutes map[string]RouteInfo
}

var newHandle bool
var currentRouteInfo RouteInfo

// getAllRouteInfo generates the api documentation
func getAllRouteInfo(path string) []Doc {
	fset := token.NewFileSet() // positions are relative to fset
	dm, err := parser.ParseDir(fset, path, filterFile, parser.ParseComments)
	if err != nil {
		sdk.Exit(err.Error())
	}

	// Get all method and documentation
	var methods map[string]Method
	for _, f := range dm {
		p := godoc.New(f, "./", 1)
		for _, t := range p.Types {
			if t.Name == "API" {
				methods = make(map[string]Method)
				for _, f := range t.Methods {
					methods[f.Name] = Method{
						Name: f.Name,
						Doc:  f.Doc,
					}
				}
				break
			}
		}
	}

	// Get all Handlers
	d, err := parser.ParseFile(fset, path+"/api_routes.go", nil, parser.AllErrors)
	v := newVisitor()
	ast.Walk(v, d)

	allDocs := []Doc{}
	for _, m := range methods {
		routeInfo, ok := v.allRoutes[m.Name]
		if !ok {
			continue
		}
		d := Doc{}
		extractFromMethod(&d, m)
		extractFromRouteInfo(&d, routeInfo)
		allDocs = append(allDocs, d)
	}

	sort.Slice(allDocs, func(i, j int) bool { return allDocs[i].URL < allDocs[j].URL })

	return allDocs
}

func writeRouteInfo(docs []Doc, genPath string) error {
	f, err := os.Create(genPath + "/_index.md")
	if err != nil {
		return err
	}
	defer f.Close()

	content := `+++
title = "REST API"
+++

## Routes
`

	if _, err := f.WriteString(content); err != nil {
		return err
	}

	var currentTitle string
	for _, doc := range docs {
		var content, lineTitle string

		t := strings.Split(doc.URL, "/")
		if len(t) >= 2 {
			lineTitle = t[1]
		}

		if currentTitle != lineTitle {
			content = fmt.Sprintf("## %s\n", lineTitle)
		}
		currentTitle = lineTitle
		content += fmt.Sprintf("#### %s `%s`\n\n", doc.HTTPOperation, doc.URL)
		for _, v := range doc.QueryParams {
			content += fmt.Sprintf("* QueryParam: %s\n", v)
		}

		if doc.Title != "" {
			content += fmt.Sprintf("* Title: %s\n", doc.Title)
		}

		if doc.Description != "" {
			content += fmt.Sprintf("* Description: %s\n", doc.Description)
		}

		content += fmt.Sprintf("* Method: [%s](https://github.com/ovh/cds/search?q=%%22func+%%28api+*API%%29+%s%%22)\n", doc.Method, doc.Method)
		if len(doc.Middleware) > 0 {
			content += fmt.Sprintf("* Middleware(s): ")
		}
		for _, v := range doc.Middleware {
			content += fmt.Sprintf("%s: %s\n", v.Name, v.Value)
		}

		if doc.Body != "" {
			content += fmt.Sprintf("* Body: \n\n```\n%s\n```\n", doc.Body)
		}

		content += "\n\n"

		if _, err := f.WriteString(content); err != nil {
			return err
		}
	}
	f.Sync()
	return nil
}

func extractFromMethod(doc *Doc, m Method) {
	docSliptted := strings.Split(m.Doc, "\n")
	for _, dLine := range docSliptted {
		if strings.Contains(dLine, title) {
			doc.Title = strings.Trim(strings.Replace(dLine, title, "", -1), " ")
		}
		if strings.Contains(dLine, description) {
			doc.Description = strings.Trim(strings.Replace(dLine, description, "", -1), " ")
		}
		if strings.Contains(dLine, queryParam) {
			doc.QueryParams = append(doc.QueryParams, strings.Trim(strings.Replace(dLine, queryParam, "", -1), " "))
		}
		if strings.Contains(dLine, body) {
			doc.Body = strings.Trim(strings.Replace(dLine, body, "", -1), " ")
		}
	}
}
func extractFromRouteInfo(doc *Doc, routeInfo RouteInfo) {
	doc.Method = routeInfo.Method
	doc.HTTPOperation = routeInfo.HTTPOperation
	doc.Middleware = routeInfo.Middleware
	doc.URL = buildURL(routeInfo.URL)
}

func buildURL(url string) string {
	url = strings.Replace(url, "\"", "", -1)
	urlSplitted := strings.Split(url, "/")
	for i, u := range urlSplitted {
		u = strings.Replace(strings.Replace(u, "{", "<", 1), "}", ">", 1)
		switch u {
		case "<key>", "<permProjectKey>":
			u = "<project-key>"
		case "<app>", "<permApplicationName>":
			u = "<application-name>"
		case "<pip>", "<permPipelineKey>":
			u = "<pipeline-name>"
		case "<permWorkflowName>":
			u = "<workflow-name>"
		case "<permEnvironmentName>":
			u = "<environment-name>"
		case "<permID>":
			u = "<token>"
		case "<permGroupName>", "<groupName>":
			u = "<group-name>"
		case "<nodeRunID>":
			u = "node-run-id"
		case "<user>":
			u = "<user-name>"
		}
		urlSplitted[i] = u
	}
	return strings.Join(urlSplitted, "/")
}

func filterFile(f os.FileInfo) bool {
	if strings.HasSuffix(f.Name(), "_test.go") {
		return false
	}
	return true
}

func newVisitor() visitor {
	return visitor{
		allRoutes: make(map[string]RouteInfo),
	}
}

func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch d := n.(type) {
	case *ast.SelectorExpr:
		if d.Sel.Name == "Handle" {
			// Save previous operation
			if currentRouteInfo.Method != "" && currentRouteInfo.HTTPOperation != "" {
				v.allRoutes[currentRouteInfo.Method] = currentRouteInfo
			}
			currentRouteInfo = RouteInfo{}
		} else if isHttpOperation(d.Sel.Name) {
			// new Route
			if currentRouteInfo.HTTPOperation != "" && currentRouteInfo.Method != "" {
				v.allRoutes[currentRouteInfo.Method] = currentRouteInfo
			}
			url := currentRouteInfo.URL
			currentRouteInfo = RouteInfo{
				URL:           url,
				HTTPOperation: d.Sel.Name,
			}
		} else {
			currentRouteInfo.Method = d.Sel.Name
		}
	case *ast.CallExpr:
		setMiddleWare(d)
	case *ast.BasicLit:
		if currentRouteInfo.URL == "" {
			currentRouteInfo.URL = d.Value
		}
	}
	return v
}

func isHttpOperation(op string) bool {
	switch op {
	case "GET", "POST", "PUT", "DELETE", "POSTEXECUTE":
		return true
	}
	return false
}

func setMiddleWare(callExpr *ast.CallExpr) {
	switch fmt.Sprintf("%s", callExpr.Fun) {
	case "NeedAdmin", "NeedUsernameOrAdmin", "Auth", "NeedHatchery", "NeedService", "NeedWorker", "AllowServices":
		m := Middleware{
			Name:  fmt.Sprintf("%s", callExpr.Fun),
			Value: make([]string, len(callExpr.Args)),
		}
		for i, a := range callExpr.Args {
			m.Value[i] = fmt.Sprintf("%s", a)
		}
		currentRouteInfo.Middleware = append(currentRouteInfo.Middleware, m)
	}
}
