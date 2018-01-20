package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var instance *tat.Client

// Client return a new Tat Client
func Client() *tat.Client {
	ReadConfig()
	if instance != nil {
		return instance
	}

	tc, err := tat.NewClient(tat.Options{
		URL:                   viper.GetString("url"),
		Username:              viper.GetString("username"),
		Password:              viper.GetString("password"),
		Referer:               "tatcli.v." + tat.Version,
		SSLInsecureSkipVerify: viper.GetBool("sslInsecureSkipVerify"),
	})

	if err != nil {
		log.Fatalf("Error while create new Tat Client: %s", err)
	}

	tat.DebugLogFunc = log.Debugf

	if Debug {
		tat.IsDebug = true
	}

	return tc
}

// GetSkipLimit gets skip and limit in args array
// default skip to 0 and limit to 10
func GetSkipLimit(args []string) (int, int) {
	skip := "0"
	limit := "10"
	if len(args) == 3 {
		skip = args[1]
		limit = args[2]
	} else if len(args) == 2 {
		skip = args[0]
		limit = args[1]
	}
	s, e1 := strconv.Atoi(skip)
	Check(e1)
	l, e2 := strconv.Atoi(limit)
	Check(e2)
	return s, l
}

func getJSON(s []byte) string {
	if Pretty {
		var out bytes.Buffer
		json.Indent(&out, s, "", "  ")
		return out.String()
	}
	return string(s)
}

// Print prints json return
func Print(v interface{}) {
	switch v.(type) {
	case []byte:
		fmt.Printf("%s", v)
	default:
		out, err := tat.Sprint(v)
		Check(err)
		fmt.Print(getJSON(out))
	}
}

// Check checks error, if != nil, throw panic
func Check(e error) {
	if e != nil {
		if ShowStackTrace {
			panic(e)
		} else {
			log.Fatalf("%s", e)
		}
	}
}
