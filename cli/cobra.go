package cli

import (
	"strings"

	"fmt"
	"os"

	"sort"

	"reflect"

	"github.com/spf13/cobra"
)

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

func NewCommand(c Command, run RunFunc, subCommands []*cobra.Command, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

func NewGetCommand(c Command, run RunGetFunc, subCommands []*cobra.Command, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

func NewListCommand(c Command, run RunListFunc, subCommands []*cobra.Command, mod ...CommandModifier) *cobra.Command {
	return newCommand(c, run, subCommands, mod...)
}

func newCommand(c Command, run interface{}, subCommands []*cobra.Command, mods ...CommandModifier) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = c.Name

	sort.Sort(orderArgs(c.Args...))
	sort.Sort(orderArgs(c.OptionnalArgs...))

	for _, a := range c.Args {
		cmd.Use = cmd.Use + " " + strings.ToUpper(a.Name)
	}
	for _, a := range c.OptionnalArgs {
		cmd.Use = cmd.Use + " [" + strings.ToUpper(a.Name) + "]"
	}

	for _, f := range c.Flags {
		_ = cmd.Flags().StringP(f.Name, f.ShortHand, f.Default, f.Usage)
	}

	if run != nil {
		for _, mod := range mods {
			mod(&c, run)
		}
	}

	definedArgs := append(c.Args, c.OptionnalArgs...)
	sort.Sort(orderArgs(definedArgs...))

	cmd.Short = c.Short
	cmd.Long = c.Long
	cmd.Run = func(cmd *cobra.Command, args []string) {
		//Command must receive as leat mandatory args
		if len(c.Args) > len(args) {
			ExitOnError(ErrWrongUsage, cmd.Help)
		}
		//If there is no optionnal args but there more args than expected
		if len(c.OptionnalArgs) == 0 && len(args) > len(c.Args) {
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
			var err error
			s := c.Flags[i].Name
			vals[s], err = cmd.Flags().GetString(s)
			ExitOnError(err)
			if c.Flags[i].IsValid != nil && !c.Flags[i].IsValid(vals[s]) {
				fmt.Printf("%s is invalid\n", s)
				ExitOnError(ErrWrongUsage, cmd.Help)
			}
		}

		switch f := run.(type) {
		case RunFunc:
			if f == nil {
				cmd.Help()
				os.Exit(0)
			}
			ExitOnError(f(vals), cmd.Help)
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
			fmt.Println(i)
		case RunListFunc:
			if f == nil {
				cmd.Help()
				os.Exit(0)
			}

			filter, _ := cmd.Flags().GetString("filter")
			var filters map[string]string
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
			for _, i := range s {
				fmt.Printf("%v\n", i)
			}
		default:
			panic(fmt.Errorf("Unknown function type: %T", f))
		}

	}

	cmd.AddCommand(subCommands...)

	return cmd
}

func listItem(i interface{}, filters map[string]string, quiet bool, fields []string) map[string]string {
	res := map[string]string{}

	var s reflect.Value
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		s = reflect.ValueOf(i).Elem()
	} else {
		s = reflect.ValueOf(i)
	}

	t := reflect.TypeOf(i)

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		structField := t.Field(i)
		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}
		switch f.Kind() {
		case reflect.Struct, reflect.Array, reflect.Slice, reflect.Map:
			continue
		default:
			if s.IsValid() && s.CanInterface() {
				tag, ok := structField.Tag.Lookup("cli")
				fmt.Println(tag)

			}
		}
	}
}
