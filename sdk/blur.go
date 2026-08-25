package sdk

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func NewBlur(secrets []string) (*Blur, error) {
	var alternatives []string
	for i := range secrets {
		as, err := secretAlternatives(secrets[i])
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, as...)
	}

	// Longest alternatives first: strings.Replacer gives the priority to the pattern given first,
	// so a longer alternative always wins over a shorter one matching at the same position. Without
	// this a partial match could blur only the beginning of an encoded secret.
	sort.Slice(alternatives, func(i, j int) bool { return len(alternatives[i]) > len(alternatives[j]) })

	oldNew := make([]string, 0, 2*len(alternatives))
	known := make(map[string]struct{}, len(alternatives))
	for i := range alternatives {
		// The minimum applies to each form, so a short one — a line of a multi-line secret, the
		// last wrapped line of a base64 — is not blurred on its own.
		if len(alternatives[i]) < SecretMinLength {
			continue
		}
		if _, ok := known[alternatives[i]]; ok {
			continue
		}
		known[alternatives[i]] = struct{}{}
		oldNew = append(oldNew, alternatives[i], PasswordPlaceholder)
	}

	return &Blur{
		replacer: strings.NewReplacer(oldNew...),
	}, nil
}

// secretAlternatives returns the forms of a secret that could be found in logs. Forms shorter than
// SecretMinLength are dropped by NewBlur, so a short one cannot blur unrelated text.
func secretAlternatives(v string) ([]string, error) {
	// A secret stored with surrounding blank chars is often used without them
	values := []string{v}
	if trimmed := strings.TrimSpace(v); trimmed != v {
		values = append(values, trimmed)
	}

	var res []string
	for _, value := range values {
		as, err := encodingAlternatives(value)
		if err != nil {
			return nil, err
		}
		res = append(res, as...)
	}

	// Logs are blurred line by line, so every line of a multi line secret, like a private key, has
	// to be an alternative on its own: the whole secret never appears in a single log line.
	if strings.ContainsAny(v, "\r\n") {
		for _, line := range strings.FieldsFunc(v, func(r rune) bool { return r == '\n' || r == '\r' }) {
			res = append(res, line, strings.TrimSpace(line))
		}
	}

	return res, nil
}

// encodingAlternatives returns the escapes and the encodings of a value that jobs commonly print.
func encodingAlternatives(v string) ([]string, error) {
	jsonAlternative, err := json.Marshal(v)
	if err != nil {
		return nil, WithStack(err)
	}
	jsonAlt := buildJsonAlternative(v)
	res := []string{
		v,
		strings.Replace(v, "'", `'"'"'`, -1), // Useful to match secrets from 'env' in script steps

		// The worker exports every environment variable through OneLineValue, so a multi-line
		// secret reaches a step as a single line holding the two chars `\n`. The JSON escape below
		// covers it only as long as the secret holds nothing else JSON escapes.
		OneLineValue(v),
		strings.Trim(string(jsonAlternative), "\""),
		url.QueryEscape(v), // Blank chars encoded as '+', as in a query string
		url.PathEscape(v),  // Blank chars encoded as '%20', as in an URL path or a basic auth

		// Markup escaping comes in flavours: the quotes are not always escaped, and not always
		// as a numeric reference. All of them give back the value itself when it holds no markup
		// char, and are then dropped as duplicates.
		html.EscapeString(v),
		markupEscape(v, ""),
		markupEscape(v, "xml"),

		// Hex encoding does not need any end of line alternative as it encodes each byte on its
		// own: the encoding of the secret is always a prefix of the encoding of the secret + '\n'
		hex.EncodeToString([]byte(v)),
		strings.ToUpper(hex.EncodeToString([]byte(v))),

		// Useful to match a JSON secret printed through the toJson function, which spaces out its
		// punctuation, and the same value without its outer braces
		jsonAlt,
		strings.TrimSuffix(strings.TrimPrefix(jsonAlt, "{ "), "}"),
	}
	return append(res, base64Alternatives(v)...), nil
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
	case reflect.Int:
		data := e.Interface().(int)
		if b.String(strconv.Itoa(data)) == PasswordPlaceholder {
			e.SetInt(0)
		}
	default:
		return fmt.Errorf("cannot blur given value of type %q", v.Type())
	}

	return nil
}

// markupEscape escapes the markup chars of a value the way most HTML and XML escapers do:
// only the three structural chars, and with the named quote references of XML when asked.
func markupEscape(v string, flavour string) string {
	replacements := []string{"&", "&amp;", "<", "&lt;", ">", "&gt;"}
	if flavour == "xml" {
		replacements = append(replacements, `"`, "&quot;", "'", "&apos;")
	}
	return strings.NewReplacer(replacements...).Replace(v)
}

// base64WrapWidth is the line width used by the base64 command line tool when -w is not given.
const base64WrapWidth = 76

// base64Alternatives returns the base64 encodings of a secret that can be found in logs. All the
// standard alphabets are covered, padded and unpadded, because the encoding used by the job is
// unknown. Trailing end of lines are added to the secret before encoding as shell snippets usually
// rely on `echo "$secret" | base64` and echo appends an end of line to its argument: that extra
// byte changes the tail of the encoded value so the encoding of the raw secret does not match it.
func base64Alternatives(v string) []string {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var res []string
	for _, input := range []string{v, v + "\n", v + "\r\n"} {
		for _, enc := range encodings {
			res = append(res, enc.EncodeToString([]byte(input)))
		}
		// The base64 command line tool wraps its output every base64WrapWidth chars unless -w0 is
		// given. As logs are blurred line by line, every wrapped line must be an alternative.
		res = append(res, splitLength(base64.StdEncoding.EncodeToString([]byte(input)), base64WrapWidth)...)
	}
	return append(res, base64EmbeddedAlternatives(v)...)
}

// base64EmbeddedAlternatives returns what stays stable of the base64 of a secret when it is encoded
// as a part of something larger, as in a registry credential file, where `base64("user:$password")`
// never holds the encoding of the password on its own.
//
// Base64 reads three bytes at a time, so what a prefix changes is only the alignment: dropping the
// first one or two bytes of the secret gives back the alignment of a prefix of any length. What
// follows the secret only changes the last chars of the encoding, which are dropped for the same
// reason.
//
// What is left out stays readable in the logs, a few chars around the blurred value: at most the
// two first bytes of the secret, and nothing of what follows it, as the bits of the bytes it
// shares with its suffix are split between a blurred char and a readable one. The padding is
// dropped with them, its length telling the length of the secret modulo three.
func base64EmbeddedAlternatives(v string) []string {
	encodings := []*base64.Encoding{base64.RawStdEncoding, base64.RawURLEncoding}
	res := make([]string, 0, 3*len(encodings))
	for shift := 0; shift < 3 && shift < len(v); shift++ {
		aligned := v[shift:]
		for _, enc := range encodings {
			encoded := enc.EncodeToString([]byte(aligned))
			// A secret whose length is not a multiple of three ends on an incomplete group of
			// three bytes, encoded together with whatever follows it in the stream
			if len(aligned)%3 != 0 {
				encoded = encoded[:len(encoded)-1]
			}
			res = append(res, encoded)
		}
	}
	return res
}

// splitLength cuts s in chunks of at most width chars, it returns nothing if s is not longer.
func splitLength(s string, width int) []string {
	if len(s) <= width {
		return nil
	}
	var res []string
	for len(s) > width {
		res = append(res, s[:width])
		s = s[width:]
	}
	if len(s) > 0 {
		res = append(res, s)
	}
	return res
}

func buildJsonAlternative(v string) string {
	return strings.Replace(
		strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						strings.Replace(
							strings.Replace(v, "\n", "", -1),
							":", ": ", -1),
						"{", "{ ", -1),
					"}", " }", -1),
				"[", "[ ", -1),
			"]", " ]", -1),
		",", ", ", -1)
}
