package log

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
)

// CDSFormatter ...
type CDSFormatter struct{}

// Format format a log
func (f *CDSFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var keys = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		if k != "prefix" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	b := &bytes.Buffer{}
	prefixFieldClashes(entry.Data)
	f.printColored(b, entry, keys)

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *CDSFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string) {
	var levelColor string
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = ansi.Green
	case logrus.WarnLevel:
		levelColor = ansi.Yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = ansi.Red
	default:
		levelColor = ansi.Blue
	}

	if entry.Level != logrus.WarnLevel {
		levelText = strings.ToUpper(entry.Level.String())
	} else {
		levelText = "WARN"
	}
	levelText = "[" + levelText + "]"

	fmt.Fprintf(b, "%s %s%+5s%s %s", entry.Time.Format("2006-01-02 15:04:05"), levelColor, levelText, ansi.Reset, entry.Message)
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " %s%s%s=%+v", levelColor, k, ansi.Reset, v)
	}
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func (f *CDSFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	b.WriteString(key)
	b.WriteByte('=')

	switch value := value.(type) {
	case string:
		if needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	default:
		fmt.Fprint(b, value)
	}
	b.WriteByte(' ')
}

func prefixFieldClashes(data logrus.Fields) {
	if _, ok := data["time"]; ok {
		data["fields.time"] = data["time"]
	}
	if _, ok := data["msg"]; ok {
		data["fields.msg"] = data["msg"]
	}
	if _, ok := data["level"]; ok {
		data["fields.level"] = data["level"]
	}
}
