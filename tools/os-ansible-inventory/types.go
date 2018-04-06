package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

// Host is a map of HostVar, keyed by a hostname.
type Host struct {
	Name    string
	Address string
	Extra   map[string]string
}

// Group is struct that contains a list of hosts.
type Group struct {
	Hosts []Host `yaml:"hosts,omitempty"`
}

// Groups is a map of groups, keyed by a group name.
type Groups map[string]Group

// Inventory represents an Ansible Inventory consists of host variables and groups.
type Inventory struct {
	Groups Groups `json:"groups,omitempty"`
}

// Filter represent a filter on server matadata
type Filter struct {
	Key, Operator, Value string
}

// Match checks if a key, valur pair matches with the filter
func (f Filter) Match(k, v string) (bool, error) {
	switch f.Operator {
	case "=":
		if strings.ToUpper(k) == strings.ToUpper(f.Key) && strings.ToUpper(v) == strings.ToUpper(f.Value) {
			return true, nil
		}
	case "~":
		if strings.ToUpper(k) == strings.ToUpper(f.Key) {
			r, err := regexp.Compile(f.Value)
			if err != nil {
				return false, err
			}
			return r.Match([]byte(v)), nil
		}
	default:
		return false, fmt.Errorf("unsupported operator %s", f.Operator)
	}
	return false, nil
}

// ServerList is a list of server
type ServerList struct {
	l   []servers.Server
	err error
}

// Filter  a list of server
func (l *ServerList) Filter(f Filter) {
	if l.err != nil {
		fmt.Println(l.err)
		return
	}

	list := []servers.Server{}
	for _, s := range l.l {
		for k, v := range s.Metadata {
			ok, err := f.Match(k, v)
			if err != nil {
				l.err = err
				return
			}
			if ok {
				list = append(list, s)
			}
		}
	}
	l.l = list
}
