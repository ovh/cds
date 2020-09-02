package cli

import (
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
		fmt.Printf("Error(request_id:%s): %s\n", e.RequestID, e.Message)
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

// SubCommands represents an array of cobra.Command
type SubCommands []*cobra.Command

// NewCommand creates a new cobra command with or without a RunFunc and eventually subCommands
func NewCommand(c Command, run RunFunc, subCommands SubCommands, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

// NewGetCommand creates a new cobra command with a RunGetFunc and eventually subCommands
func NewGetCommand(c Command, run RunGetFunc, subCommands SubCommands, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

// NewDeleteCommand creates a new cobra command with a RunDeleteFunc and eventually subCommands
func NewDeleteCommand(c Command, run RunDeleteFunc, subCommands SubCommands, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

// NewListCommand creates a new cobra command with a RunListFunc and eventually subCommands
func NewListCommand(c Command, run RunListFunc, subCommands SubCommands, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

func newCommand(c Command, run interface{}, subCommands SubCommands, mods ...CommandModifier) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOutput(os.Stdout)
	cmd.Use = c.Name

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
				ExitOnError(err)
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
		c, _ := cmd.Flags().GetString("context")
		n, _ := cmd.Flags().GetBool("no-interactive")
		b, _ := cmd.Flags().GetBool("insecure")
		v, _ := cmd.Flags().GetBool("verbose")
		f, _ := cmd.Flags().GetString("file")
		vals["context"] = append(vals["context"], c)
		vals["file"] = append(vals["file"], f)
		vals["no-interactive"] = append(vals["no-interactive"], fmt.Sprintf("%v", n))
		vals["insecure"] = append(vals["insecure"], fmt.Sprintf("%v", b))
		vals["verbose"] = append(vals["verbose"], fmt.Sprintf("%v", v))

		format, _ := cmd.Flags().GetString("format")

		switch f := run.(type) {
		case RunFunc:
			if f == nil {
				cmd.Help() // nolint
				OSExit(0)
			}
			ExitOnError(f(vals))
			OSExit(0)
		case RunGetFunc:
			if f == nil {
				cmd.Help() // nolint
				OSExit(0)
			}
			i, err := f(vals)
			if err != nil {
				ExitOnError(err)
			}

			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")
			fields, _ := cmd.Flags().GetString("fields")
			var fs []string
			if fields != "" {
				fs = strings.Split(fields, ",")
			}
			i = listItem(i, nil, quiet, fs, verbose, map[string]string{})
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
				if quiet {
					fmt.Println(i.(map[string]string)["key"])
					return
				}
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 10, 0, 1, ' ', 0)
				m, err := dump.ToStringMap(i)
				ExitOnError(err)

				itemKeys := []string{}
				for k := range m {
					itemKeys = append(itemKeys, k)
				}

				sort.Strings(itemKeys)

				for _, k := range itemKeys {
					fmt.Fprintln(w, k+"\t"+m[k])
				}
				w.Flush()
				return
			}

		case RunListFunc:
			if f == nil {
				cmd.Help() // nolint
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
					if len(s) != 2 {
						ExitOnError(fmt.Errorf("Filter should be formatted like name=value"))
					}
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
				cmd.Help() // nolint
				OSExit(0)
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force && !AskConfirm("Are you sure to delete?") {
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

	if s.Kind() == reflect.Map {
		m, _ := dump.ToStringMap(i)
		return m
	}

	if s.Kind() != reflect.Struct {
		return nil
	}

	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = reflect.TypeOf(reflect.ValueOf(i).Elem().Interface())
	}

	for i := 0; i < s.NumField(); i++ {
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
				tag := structField.Tag.Get("cli")
				if tag == "-" {
					continue
				}

				var isKey bool
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

				// if there are filters and current tag value not match return nil item
				if len(filters) > 0 {
					for k, v := range filters {
						// filter only if tag match
						matchTag := k == tag || strings.ToUpper(k) == strings.ToUpper(tag)
						if !matchTag {
							continue
						}

						// transform filter to regex
						if !strings.HasPrefix(v, "^") {
							v = "^" + v
						}
						if !strings.HasSuffix(v, "$") {
							v = v + "$"
						}

						// if the value don't match the item will not be displayed
						matchValue, err := regexp.MatchString(v, fmt.Sprintf("%v", f.Interface()))
						if err != nil {
							panic(err)
						}
						if !matchValue {
							return nil
						}
					}
				}

				// if there are fields list, add only tag that match in result (ignore for quiet mode)
				if !quiet && len(fields) > 0 {
					var visible bool
					for _, ff := range fields {
						matchTag := ff == tag || strings.ToUpper(ff) == tag || strings.ToLower(ff) == tag
						if matchTag {
							visible = true
							break
						}
					}
					if !visible {
						continue
					}
				}

				// if not quiet mode add the key:value to result else if quiet add only the key
				if !quiet {
					if !f.IsValid() {
						res[tag] = ""
					} else {
						res[tag] = fmt.Sprintf("%v", f.Interface())
					}
				} else if isKey {
					res["key"] = fmt.Sprintf("%v", f.Interface())
				}
			}
		}
	}
	return res
}
