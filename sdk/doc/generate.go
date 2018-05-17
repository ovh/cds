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
	"text/template"

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
		// @requestBody {"mykey": "myval"}
		// @responseBody {"mykey": "myval"}
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
	title        = "@title"
	description  = "@description"
	queryParam   = "@params"
	requestBody  = "@requestBody"
	responseBody = "@responseBody"
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
	ResponseBody  string
	RequestBody   string
	Middleware    []Middleware
	HTTPOperation string
}

// RouteInfo Information on a route
type RouteInfo struct {
	URL           string
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

type pageTmpl struct {
	Title  string
	Routes []routeTmpl
}

type routeTmpl struct {
	Title        string
	Description  string
	URL          string
	Method       string
	Permissions  string
	QueryParams  []string
	Code         string
	RequestBody  string
	ResponseBody string
}

func writeRouteInfo(inputDocs []Doc, genPath string) error {
	docsSection := make(map[string][]Doc)
	for _, doc := range inputDocs {
		t := strings.Split(doc.URL, "/")
		if len(t) >= 2 {
			lineTitle := t[1]
			docsSection[lineTitle] = append(docsSection[lineTitle], doc)
		}
	}

	for name, docs := range docsSection {
		filename := fmt.Sprintf("%s/%s.md", genPath, name)
		if _, err := os.Stat(filename); err == nil {
			if err := os.Remove(filename); err != nil {
				return err
			}
		}
		fmt.Printf("create file %s\n", filename)
		f, err := os.Create(filename)
		if err != nil {
			return err
		}

		t := template.New("routes")
		t, err = t.ParseFiles("sdk/doc/routes.tmpl")
		if err != nil {
			return err
		}
		dataPage := pageTmpl{
			Title:  name,
			Routes: []routeTmpl{},
		}

		for _, doc := range docs {
			route := routeTmpl{}
			if doc.Title == "" {
				route.Title = fmt.Sprintf("%s `%s`", doc.HTTPOperation, doc.URL)
			} else {
				route.Title = doc.Title
			}

			var permissions string
			var noAuth bool
			for _, v := range doc.Middleware {
				if permissions != "" {
					permissions += " - "
				}
				permissions += fmt.Sprintf(" %s: %s", v.Name, strings.Join(v.Value, ","))
				if v.Name == "Auth" {
					for _, value := range v.Value {
						if value == "false" {
							noAuth = true
						}
					}
				}
			}

			if !noAuth {
				if permissions != "" {
					permissions += " - "
				}
				permissions += " Auth: true"
			}
			route.Permissions = permissions
			route.URL = doc.URL
			route.Method = doc.HTTPOperation
			route.QueryParams = doc.QueryParams
			route.Code = fmt.Sprintf("[%s](https://github.com/ovh/cds/search?q=%%22func+%%28api+*API%%29+%s%%22)\n", doc.Method, doc.Method)
			route.Description = doc.Description
			route.RequestBody = doc.RequestBody
			route.ResponseBody = doc.ResponseBody
			dataPage.Routes = append(dataPage.Routes, route)
		}

		if err := t.ExecuteTemplate(f, "routes.tmpl", dataPage); err != nil {
			return err
		}
		f.Sync()
		f.Close()
	}
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
		if strings.Contains(dLine, requestBody) {
			doc.RequestBody = strings.Trim(strings.Replace(dLine, requestBody, "", -1), " ")
		}
		if strings.Contains(dLine, responseBody) {
			doc.ResponseBody = strings.Trim(strings.Replace(dLine, responseBody, "", -1), " ")
		}
	}
	if len(docSliptted) > 0 && doc.Title == "" {
		doc.Title = strings.Replace(docSliptted[0], "Handler", "", 1)
		doc.Title = strings.Replace(doc.Title, "Handle", "", 1)
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
		} else if isHTTPOperation(d.Sel.Name) {
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

func isHTTPOperation(op string) bool {
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
