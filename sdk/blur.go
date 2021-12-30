package sdk

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

func NewBlur(secrets []string) (*Blur, error) {
	alternativeFuncs := func(v string) ([]string, error) {
		jsonAlternative, err := json.Marshal(v)
		if err != nil {
			return nil, WithStack(err)
		}
		return []string{
			v,
			strings.Replace(v, "'", `'"'"'`, -1), // Useful to match secrets from 'env' in script steps
			strings.Trim(string(jsonAlternative), "\""),
			url.QueryEscape(v),
			base64.StdEncoding.EncodeToString([]byte(v)),
		}, nil
	}

	var alternatives []string
	for i := range secrets {
		as, err := alternativeFuncs(secrets[i])
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, as...)
	}
	sort.Slice(alternatives, func(i, j int) bool { return len(alternatives[i]) > len(alternatives[j]) })

	oldNew := make([]string, 0, 2*len(alternatives))
	for i := range alternatives {
		if len(alternatives[i]) >= SecretMinLength {
			oldNew = append(oldNew, alternatives[i], PasswordPlaceholder)
		}
	}

	return &Blur{
		replacer: strings.NewReplacer(oldNew...),
	}, nil
}

type Blur struct {
	replacer *strings.Replacer
}

func (b *Blur) String(s string) string {
	return b.replacer.Replace(s)
}

func (b *Blur) Interface(i interface{}) error {
	v := reflect.ValueOf(i)
	e := v.Elem()

	switch e.Kind() {
	case reflect.Slice:
		for i := 0; i < e.Len(); i++ {
			if err := b.Interface(e.Index(i).Addr().Interface()); err != nil {
				return err
			}
		}
	case reflect.String:
		data := e.Interface().(string)
		e.SetString(b.String(data))
	case reflect.Struct:
		for i := 0; i < e.NumField(); i++ {
			if err := b.Interface(e.Field(i).Addr().Interface()); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("cannot blur given value of type %q", v.Type())
	}

	return nil
}
