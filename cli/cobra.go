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

	"github.com/ovh/cds/sdk"
)

// ShellMode will os.Exit if false, display only exit code if true
var ShellMode bool

//ExitOnError if the error is not nil; exit the process with printing help functions and the error
func ExitOnError(err error, helpFunc ...func() error) {
	if err == nil {
		return
	}

	code := 50 // default error code

	switch e := err.(type) {
	case sdk.Error:
		fmt.Printf("Error(%s): %s\n", e.UUID, e.Message)
	case *Error:
		code = e.Code
		fmt.Println("Error:", e.Error())
	default:
		fmt.Println("Error:", err.Error())
	}

	for _, f := range helpFunc {
		f()
	}

	OSExit(code)
}

// OSExit will os.Exit if ShellMode is false, display only exit code if true
func OSExit(code int) {
	if ShellMode {
		// display code only if os.Exit is not ok
		if code != 0 {
			fmt.Printf("Command exit with code %d\n", code)
		}
	} else {
		os.Exit(code)
	}
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
	cmd.SetOutput(os.Stdout)
	cmd.Use = c.Name

	sort.Sort(orderArgs(c.Args...))
	sort.Sort(orderArgs(c.OptionalArgs...))

	if len(c.Ctx) > 0 {
		cmd.Use = cmd.Use + " ["
	}

	for _, a := range c.Ctx {
		cmd.Use = cmd.Use + " " + strings.ToUpper(a.Name)
	}

	if len(c.Ctx) > 0 {
		cmd.Use = cmd.Use + " ]"
	}

	for _, a := range c.Args {
		cmd.Use = cmd.Use + " " + strings.ToUpper(a.Name)
	}
	for _, a := range c.OptionalArgs {
		cmd.Use = cmd.Use + " [" + strings.ToUpper(a.Name) + "]"
	}
	if c.VariadicArgs.Name != "" {
		cmd.Use = cmd.Use + " " + strings.ToUpper(c.VariadicArgs.Name) + " ..."
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
		switch f.Type {
		case FlagBool:
			b, _ := strconv.ParseBool(f.Default)
			_ = cmd.Flags().BoolP(f.Name, f.ShortHand, b, f.Usage)
		case FlagSlice:
			_ = cmd.Flags().StringSliceP(f.Name, f.ShortHand, nil, f.Usage)
		case FlagArray:
			_ = cmd.Flags().StringArrayP(f.Name, f.ShortHand, nil, f.Usage)
		default:
			_ = cmd.Flags().StringP(f.Name, f.ShortHand, f.Default, f.Usage)
		}
	}

	definedArgs := append(c.Ctx, c.Args...)
	definedArgs = append(definedArgs, c.OptionalArgs...)
	sort.Sort(orderArgs(definedArgs...))
	definedArgs = append(definedArgs, c.VariadicArgs)

	cmd.Short = c.Short
	cmd.Long = c.Long
	cmd.Hidden = c.Hidden
	cmd.Example = c.Example
	cmd.AddCommand(subCommands...)

	if run == nil || reflect.ValueOf(run).IsNil() {
		cmd.Run = nil
		cmd.RunE = nil
		return cmd
	}

	var argsToVal = func(args []string) Values {
		vals := Values{}
		nbDefinedArgs := len(definedArgs)
		if c.VariadicArgs.Name != "" {
			nbDefinedArgs--
		}
		for i := range args {
			if i < nbDefinedArgs {
				s := definedArgs[i].Name
				if definedArgs[i].IsValid != nil && !definedArgs[i].IsValid(args[i]) {
					fmt.Printf("%s is invalid\n", s)
					ExitOnError(ErrWrongUsage, cmd.Help)
				}
				vals[s] = append(vals[s], args[i])
			} else {
				vals[c.VariadicArgs.Name] = append(vals[c.VariadicArgs.Name], strings.Join(args[i:], ","))
				break
			}
		}

		for i := range c.Flags {
			s := c.Flags[i].Name
			switch c.Flags[i].Type {
			case FlagBool:
				b, err := cmd.Flags().GetBool(s)
				ExitOnError(err)
				vals[s] = append(vals[s], fmt.Sprintf("%v", b))
			case FlagSlice:
				slice, err := cmd.Flags().GetStringSlice(s)
				ExitOnError(err)
				vals[s] = append(vals[s], strings.Join(slice, "||"))
			case FlagArray:
				array, err := cmd.Flags().GetStringArray(s)
				ExitOnError(err)
				vals[s] = array
			default:
				val, err := cmd.Flags().GetString(s)
				ExitOnError(err)
				vals[s] = append(vals[s], val)
			}
			if c.Flags[i].IsValid != nil {
				for _, v := range vals[s] {
					if !c.Flags[i].IsValid(v) {
						fmt.Printf("%s is invalid\n", s)
						ExitOnError(ErrWrongUsage, cmd.Help)
					}
				}
			}
		}
		return vals
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if c.PreRun != nil {
			if err := c.PreRun(&c, &args); err != nil {
				ExitOnError(ErrWrongUsage, cmd.Help)
				return
			}
		}

		//Command must receive as least mandatory args
		if len(c.Args)+len(c.Ctx) > len(args) {
			ExitOnError(ErrWrongUsage, cmd.Help)
			return
		}

		//If there is no optional args but there more args than expected
		if c.VariadicArgs.Name == "" && len(c.OptionalArgs) == 0 && (len(args) > len(c.Args)+len(c.Ctx)) {
			ExitOnError(ErrWrongUsage, cmd.Help)
			return
		}
		//If there is a variadic arg, we condider at least one arg mandatory
		if c.VariadicArgs.Name != "" && (len(args) < len(c.Args)+len(c.Ctx)+1) {
			ExitOnError(ErrWrongUsage, cmd.Help)
			return
		}

		vals := argsToVal(args)

		format, _ := cmd.Flags().GetString("format")

		switch f := run.(type) {
		case RunFunc:
			if f == nil {
				cmd.Help()
				OSExit(0)
			}
			ExitOnError(f(vals))
			OSExit(0)
		case RunGetFunc:
			if f == nil {
				cmd.Help()
				OSExit(0)
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
				if ShellMode {
					fmt.Fprint(cmd.OutOrStdout(), string(b))
				} else {
					fmt.Println(string(b))
				}
			case "yaml":
				b, err := yaml.Marshal(i)
				ExitOnError(err)
				if ShellMode {
					fmt.Fprint(cmd.OutOrStdout(), string(b))
				} else {
					fmt.Println(string(b))
				}
			default:
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 10, 0, 1, ' ', 0)
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
				OSExit(0)
			}

			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")
			filter, _ := cmd.Flags().GetString("filter")
			fields, _ := cmd.Flags().GetString("fields")
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
				var fs []string
				if fields != "" {
					fs = strings.Split(fields, ",")
				}
				item := listItem(i, filters, quiet, fs, verbose, map[string]string{})
				if len(item) == 0 {
					continue
				}

				if quiet {
					fmt.Fprintln(cmd.OutOrStdout(), item["key"])
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
				table := tablewriter.NewWriter(cmd.OutOrStdout())
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
				OSExit(0)
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force && !AskForConfirmation("Are you sure to delete?") {
				fmt.Println("Deletion aborted")
				OSExit(0)
			}

			err := f(vals)
			if err == nil {
				fmt.Println("Delete with success")
			}
			ExitOnError(err)
			OSExit(0)

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

	if s.Kind() != reflect.Struct {
		return nil
	}

	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = reflect.TypeOf(reflect.ValueOf(i).Elem().Interface())
	}

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
							if ff == tag || strings.ToUpper(ff) == tag || strings.ToLower(ff) == tag {
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
