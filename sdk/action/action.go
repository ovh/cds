package action

import (
	"fmt"
	"sort"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"

	yaml "gopkg.in/yaml.v2"
)

// List of all available actions.
var List = []Manifest{
	ArtifactDownload,
	ArtifactUpload,
	PushBuildInfo,
	CheckoutApplication,
	Coverage,
	DeployApplication,
	GitClone,
	GitTag,
	InstallKey,
	JUnit,
	Promote,
	ReleaseVCS,
	Release,
	Script,
}

// Manifest for a action.
type Manifest struct {
	Action  sdk.Action
	Example exportentities.PipelineV1
}

// Markdown returns string formatted for an action.
func (m Manifest) Markdown() string {
	var sp, rq string
	ps := m.Action.Parameters
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	for _, p := range ps {
		sp += fmt.Sprintf("* **%s**: %s\n", p.Name, p.Description)
	}
	if sp == "" {
		sp = "No Parameter"
	}

	rs := m.Action.Requirements
	sort.Slice(rs, func(i, j int) bool { return rs[i].Name < rs[j].Name })
	for _, r := range rs {
		rq += fmt.Sprintf("* **%s**: type: %s Value: %s\n", r.Name, r.Type, r.Value)
	}

	if rq == "" {
		rq = "No Requirement"
	}

	ex, _ := yaml.Marshal(m.Example)

	info := fmt.Sprintf(`---
title: "%s"
card:
  name: builtin
---

**%s** is a builtin action, you can't modify it.

%s

## Parameters

%s

## Requirements

%s

## YAML example

Example of a pipeline using %s action:
%s
`,
		m.Action.Name,
		m.Action.Name,
		m.Action.Description,
		sp,
		rq,
		m.Action.Name,
		fmt.Sprintf("```yml\n%s\n```", string(ex)))

	return info
}
