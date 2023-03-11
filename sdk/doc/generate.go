package doc

import (
	"bytes"
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

	"github.com/ovh/cds/sdk"
)

// GenerateDocumentation generates hugo documentation for a command line
func GenerateDocumentation(root *cobra.Command, genPath, gitPath string) error {
	const fmTemplate = `---
title: "%s"
notitle: true
notoc: true
---
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
		base = strings.Replace(strings.ToLower(base), "_", "/", -1)
		if rootName == base {
			return fmt.Sprintf("/docs/components/%s/", rootName)
		}
		return fmt.Sprintf("/docs/components/%s/%s/", rootName, base)
	}

	fmt.Printf("%s\n", rootName)
	if err := os.MkdirAll(genPath+"/"+rootName, os.ModePerm); err != nil {
		return err
	}

	if gitPath != "" {
		if err := os.MkdirAll(genPath+"/../../development/rest", os.ModePerm); err != nil {
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
		func (api *API) getActionsHandler() service.Handler {
		[...]
		*/
		if err := writeRouteInfo(getAllRouteInfo(gitPath+"/engine/api"), genPath+"/../../development/rest"); err != nil {
			return err
		}
	}

	return genMarkdownTreeCustom(root, genPath+"/"+rootName, filePrepender, linkHandler)
}

// genMarkdownTreeCustom is the same as GenMarkdownTree, but
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
		if c.Short != "" {
			c.Short = fmt.Sprintf("`%s`", c.Short)
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
	return GenMarkdownCustom(cmd, f, linkHandler)
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
	Middlewares   []Middleware
	Scopes        []string
	HTTPOperation string
}

// RouteInfo Information on a route
type RouteInfo struct {
	URL           string
	Method        string
	Middlewares   []Middleware
	Scopes        []string
	HTTPOperation string
}

// Middleware represents a middleware
type Middleware struct {
	Name  string
	Value []string
}

type visitor struct {
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
	if err != nil {
		sdk.Exit(err.Error())
	}

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
	Scopes       string
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

		if err := printSection(name, docs, f); err != nil {
			return err
		}

		f.Sync()
		f.Close()
	}
	return nil
}

func docTitle(doc Doc) string {
	if doc.Title == "" {
		return fmt.Sprintf("%s `%s`", doc.HTTPOperation, doc.URL)
	}
	return doc.Title
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
}
func extractFromRouteInfo(doc *Doc, routeInfo RouteInfo) {
	doc.Method = routeInfo.Method
	doc.HTTPOperation = routeInfo.HTTPOperation
	doc.Middlewares = routeInfo.Middlewares
	doc.Scopes = routeInfo.Scopes
	doc.URL = CleanURL(routeInfo.URL)
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
			scopes := currentRouteInfo.Scopes
			currentRouteInfo = RouteInfo{
				URL:           url,
				HTTPOperation: d.Sel.Name,
				Scopes:        scopes,
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
	case "NeedAdmin", "Auth":
		m := Middleware{
			Name:  fmt.Sprintf("%s", callExpr.Fun),
			Value: make([]string, len(callExpr.Args)),
		}
		for i, a := range callExpr.Args {
			m.Value[i] = fmt.Sprintf("%s", a)
		}
		currentRouteInfo.Middlewares = append(currentRouteInfo.Middlewares, m)
	case "Scope":
		for _, a := range callExpr.Args {
			if v, ok := a.(*ast.SelectorExpr); ok {
				currentRouteInfo.Scopes = append(currentRouteInfo.Scopes, strings.TrimPrefix(v.Sel.Name, "AuthConsumerScope"))
			}
		}
	}
}

// GenMarkdownCustom below is inspired from spf13/cobra. spf13/cobra is licensed under the Apache License 2.0

// GenMarkdownCustom creates custom markdown output.
func GenMarkdownCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	short := cmd.Short
	long := cmd.Long
	if len(long) == 0 {
		long = short
	}

	buf.WriteString("# " + name + "\n\n")
	buf.WriteString(short + "\n\n")
	buf.WriteString("## Synopsis\n\n")
	buf.WriteString(long + "\n\n")

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("## Examples\n\n")
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.Example))
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}
	if hasSeeAlso(cmd) {
		buf.WriteString("## SEE ALSO\n\n")
		if cmd.HasParent() {
			parent := cmd.Parent()
			pname := parent.CommandPath()
			link := pname + ".md"
			link = strings.Replace(link, " ", "_", -1)
			buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", pname, linkHandler(link), parent.Short))
			cmd.VisitParents(func(c *cobra.Command) {
				if c.DisableAutoGenTag {
					cmd.DisableAutoGenTag = c.DisableAutoGenTag
				}
			})
		}

		children := cmd.Commands()
		sort.Sort(byName(children))

		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			cname := name + " " + child.Name()
			link := cname + ".md"
			link = strings.Replace(link, " ", "_", -1)
			buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", cname, linkHandler(link), child.Short))
		}
		buf.WriteString("\n")
	}
	_, err := buf.WriteTo(w)
	return err
}

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasFlags() {
		buf.WriteString("## Options\n\n```\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasFlags() {
		buf.WriteString("## Options inherited from parent commands\n\n```\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n\n")
	}
	return nil
}

func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

type byName []*cobra.Command

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
