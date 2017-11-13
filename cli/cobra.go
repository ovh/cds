package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/fsamin/go-dump"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

//ExitOnError if the error is not nil; exit the process with printing help functions and the error
func ExitOnError(err error, helpFunc ...func() error) {
	if err == nil {
		return
	}

	if e, ok := err.(*Error); ok {
		fmt.Println("Error:", e.Error())
		for _, f := range helpFunc {
			f()
		}
		os.Exit(e.Code)
	}
	fmt.Println("Error:", err.Error())
	for _, f := range helpFunc {
		f()
	}
	os.Exit(50)
}

// NewCommand creates a new cobra command with or without a RunFunc and eventually subCommands
func NewCommand(c Command, run RunFunc, subCommands []*cobra.Command, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

// NewGetCommand creates a new cobra command with a RunGetFunc and eventually subCommands
func NewGetCommand(c Command, run RunGetFunc, subCommands []*cobra.Command, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

// NewDeleteCommand creates a new cobra command with a RunDeleteFunc and eventually subCommands
func NewDeleteCommand(c Command, run RunDeleteFunc, subCommands []*cobra.Command, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

// NewListCommand creates a new cobra command with a RunListFunc and eventually subCommands
func NewListCommand(c Command, run RunListFunc, subCommands []*cobra.Command, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

func newCommand(c Command, run interface{}, subCommands []*cobra.Command, mods ...CommandModifier) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = c.Name

	sort.Sort(orderArgs(c.Args...))
	sort.Sort(orderArgs(c.OptionalArgs...))

	for _, a := range c.Args {
		cmd.Use = cmd.Use + " " + strings.ToUpper(a.Name)
	}
	for _, a := range c.OptionalArgs {
		cmd.Use = cmd.Use + " [" + strings.ToUpper(a.Name) + "]"
	}

	if len(mods) == 0 {
		mods = []CommandModifier{CommandWithExtraFlags, CommandWithExtraAliases}
	}

	if run != nil {
		for _, mod := range mods {
			mod(&c, run)
		}
	}
	cmd.Aliases = c.Aliases

	for _, f := range c.Flags {
		switch f.Kind {
		case reflect.Bool:
			b, _ := strconv.ParseBool(f.Default)
			_ = cmd.Flags().BoolP(f.Name, f.ShortHand, b, f.Usage)
		default:
			_ = cmd.Flags().StringP(f.Name, f.ShortHand, f.Default, f.Usage)
		}
	}

	definedArgs := append(c.Args, c.OptionalArgs...)
	sort.Sort(orderArgs(definedArgs...))

	cmd.Short = c.Short
	cmd.Long = c.Long
	cmd.AddCommand(subCommands...)

	if run == nil || reflect.ValueOf(run).IsNil() {
		cmd.Run = nil
		cmd.RunE = nil
		return cmd
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		//Command must receive as leat mandatory args
		if len(c.Args) > len(args) {
			ExitOnError(ErrWrongUsage, cmd.Help)
		}
		//If there is no optionnal args but there more args than expected
		if len(c.OptionalArgs) == 0 && len(args) > len(c.Args) {
			ExitOnError(ErrWrongUsage, cmd.Help)
		}

		vals := Values{}
		for i := range args {
			s := definedArgs[i].Name
			if definedArgs[i].IsValid != nil && !definedArgs[i].IsValid(args[i]) {
				fmt.Printf("%s is invalid\n", s)
				ExitOnError(ErrWrongUsage, cmd.Help)
			}
			vals[s] = args[i]
		}

		for i := range c.Flags {
			s := c.Flags[i].Name
			switch c.Flags[i].Kind {
			case reflect.String:
				var err error
				vals[s], err = cmd.Flags().GetString(s)
				ExitOnError(err)
			case reflect.Bool:
				b, err := cmd.Flags().GetBool(s)
				ExitOnError(err)
				vals[s] = fmt.Sprintf("%v", b)
			}
			if c.Flags[i].IsValid != nil && !c.Flags[i].IsValid(vals[s]) {
				fmt.Printf("%s is invalid\n", s)
				ExitOnError(ErrWrongUsage, cmd.Help)
			}
		}

		format, _ := cmd.Flags().GetString("format")

		switch f := run.(type) {
		case RunFunc:
			if f == nil {
				cmd.Help()
				os.Exit(0)
			}
			ExitOnError(f(vals))
			os.Exit(0)
		case RunGetFunc:
			if f == nil {
				cmd.Help()
				os.Exit(0)
			}
			i, err := f(vals)
			if err != nil {
				ExitOnError(err)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			if !verbose {
				i = listItem(i, nil, false, nil, verbose, map[string]string{})
			}

			switch format {
			case "json":
				b, err := json.Marshal(i)
				ExitOnError(err)
				fmt.Println(string(b))
			case "yaml":
				b, err := yaml.Marshal(i)
				ExitOnError(err)
				fmt.Println(string(b))
			default:
				w := tabwriter.NewWriter(os.Stdout, 10, 0, 1, ' ', 0)
				e := dump.NewDefaultEncoder(new(bytes.Buffer))
				e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
				e.ExtraFields.DetailedMap = false
				e.ExtraFields.DetailedStruct = false
				e.ExtraFields.Len = false
				e.ExtraFields.Type = false
				m, err := e.ToStringMap(i)
				ExitOnError(err)
				for k, v := range m {
					fmt.Fprintln(w, k+"\t"+v)
				}
				w.Flush()
				return
			}

		case RunListFunc:
			if f == nil {
				cmd.Help()
				os.Exit(0)
			}

			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")
			filter, _ := cmd.Flags().GetString("filter")
			var filters = make(map[string]string)
			if filter != "" {
				t := strings.Split(filter, " ")
				for i := range t {
					s := strings.SplitN(t[i], "=", 2)
					filters[s[0]] = s[1]
				}
			}

			s, err := f(vals)
			if err != nil {
				ExitOnError(err)
			}

			tableHeader := []string{}
			tableData := [][]string{}
			var tableHeaderReady bool

			allResult := []map[string]string{}

			for _, i := range s {
				item := listItem(i, filters, quiet, nil, verbose, map[string]string{})
				if len(item) == 0 {
					continue
				}

				if quiet {
					fmt.Println(item["key"])
					continue
				}

				allResult = append(allResult, item)

				if format == "" || format == "table" {
					itemData := make([]string, len(item))
					var i int

					itemKeys := []string{}
					for k := range item {
						itemKeys = append(itemKeys, k)
					}

					sort.Strings(itemKeys)

					for _, k := range itemKeys {
						if !tableHeaderReady {
							tableHeader = append(tableHeader, strings.ToTitle(k))
						}
						itemData[i] = item[k]
						i++
					}
					tableHeaderReady = true
					tableData = append(tableData, itemData)
				}
			}

			if quiet {
				return
			}

			switch format {
			case "json":
				b, err := json.Marshal(allResult)
				ExitOnError(err)
				fmt.Println(string(b))
			case "yaml":
				b, err := yaml.Marshal(allResult)
				ExitOnError(err)
				fmt.Println(string(b))
			default:
				if len(tableData) == 0 {
					fmt.Println("nothing to display...")
					return
				}
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader(tableHeader)
				for _, v := range tableData {
					table.Append(v)
				}
				table.Render()
				return
			}

		case RunDeleteFunc:
			if f == nil {
				cmd.Help()
				os.Exit(0)
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force && !AskForConfirmation("Are you sure to delete ?") {
				fmt.Println("Deletion aborted")
				os.Exit(0)
			}

			err := f(vals)
			if err == nil {
				fmt.Println("Delete with success")
			}
			ExitOnError(err)
			os.Exit(0)

		default:
			panic(fmt.Errorf("Unknown function type: %T", f))
		}

	}

	return cmd
}

func listItem(i interface{}, filters map[string]string, quiet bool, fields []string, verbose bool, res map[string]string) map[string]string {
	var s reflect.Value
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		s = reflect.ValueOf(i).Elem()
	} else {
		s = reflect.ValueOf(i)
	}

	t := reflect.TypeOf(i)
	var ok = true
	for i := 0; i < s.NumField() && ok; i++ {
		f := s.Field(i)
		structField := t.Field(i)
		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}
		switch f.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map:
			continue
		default:
			if structField.Anonymous && f.Kind() == reflect.Struct {
				res = listItem(f.Interface(), filters, quiet, fields, verbose, res)
				continue
			}
			if s.IsValid() && s.CanInterface() {
				var isKey bool
				tag := structField.Tag.Get("cli")
				if tag == "-" {
					continue
				}

				if strings.HasSuffix(tag, ",key") {
					isKey = true
					tag = strings.Replace(tag, ",key", "", -1)
				}

				if !verbose && tag == "" {
					continue
				}

				if tag == "" {
					tag = structField.Name
				}

				if len(filters) > 0 {
					for k, v := range filters {
						if !strings.HasPrefix(v, "^") {
							v = "^" + v
						}
						if !strings.HasSuffix(v, "$") {
							v = v + "$"
						}
						match, err := regexp.MatchString(v, fmt.Sprintf("%v", f.Interface()))
						if err != nil {
							panic(err)
						}
						if k == tag && !match {
							ok = false
							continue
						}
					}
					if ok {
						res[tag] = fmt.Sprintf("%v", f.Interface())
					}
				} else {
					if quiet && isKey {
						res["key"] = fmt.Sprintf("%v", f.Interface())
						break
					}

					if len(fields) > 0 {
						for _, ff := range fields {
							if ff == tag {
								res[tag] = fmt.Sprintf("%v", f.Interface())
								continue
							}
						}
						continue
					}
					res[tag] = fmt.Sprintf("%v", f.Interface())
				}
			}
		}
	}
	return res
}
