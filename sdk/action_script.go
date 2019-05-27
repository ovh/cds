package sdk

import (
	"fmt"
	"sort"
)

// ActionInfoMarkdown returns string formatted with markdown.
func ActionInfoMarkdown(a Action, filename string) string {
	var sp, rq string
	ps := a.Parameters
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	for _, p := range ps {
		sp += fmt.Sprintf("* **%s**: %s\n", p.Name, p.Description)
	}
	if sp == "" {
		sp = "No Parameter"
	}

	rs := a.Requirements
	sort.Slice(rs, func(i, j int) bool { return rs[i].Name < rs[j].Name })
	for _, r := range rs {
		rq += fmt.Sprintf("* **%s**: type: %s Value: %s\n", r.Name, r.Type, r.Value)
	}

	if rq == "" {
		rq = "No Requirement"
	}

	info := fmt.Sprintf(`
%s

## Parameters

%s

## Requirements

%s

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/%s)

`,
		a.Description,
		sp,
		rq,
		filename)

	return info
}
