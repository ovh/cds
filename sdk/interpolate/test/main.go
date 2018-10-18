package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
)

var htmlTemplate = `{{.one.two.three}} {{.one.two.three.four}}`

type val map[string]interface{}

func (v val) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		_, _ = io.WriteString(s, fmt.Sprintf("%v", v["_"]))
	}
}

func main() {
	data := val{
		"one": val{
			"two": val{
				"three": val{
					"_":    "three-val",
					"four": "four-val",
				},
			},
		},
	}

	t := template.New("t")
	t, err := t.Parse(htmlTemplate)
	if err != nil {
		panic(err)
	}

	err = t.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}

}
