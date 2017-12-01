package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromActionScript(t *testing.T) {
	b := []byte(`
name = "TestGitClone"
description = "Clone git repository"

// Requirements
requirements = {
	"git" = {
		type = "binary"
		value = "git"
	}
	"ssh-keygen" = {
		type = "binary"
		value = "ssh-keygen"
	}
}

// Parameters
parameters = {
	 "branch" = {
		type = "string"
		value = "{{.git.branch}}"
	}
	"commit" = {
		type = "string"
		value = "{{.git.hash}}"
	}
	"directory" = {
		type = "string"
		description = "target directory"
	}
	"gitOptions" = {
		type = "string"
		description = "git clone options"
	}
	"url" = {
		type = "string"
		description = "git URL"
		value = "{{.cds.app.repo}}"
	}
}

// Steps
steps  = [{
	script = <<EOF
ssh-keygen -R github.com
EOF
    },
    {
	script = <<EOF
#! /bin/bash
set -e
echo "action git from directory"
pwd
echo "running git clone {{.gitOptions}} {{.url}} -b {{.branch}} {{.directory}}"
git clone {{.gitOptions}} {{.url}} -b {{.branch}} {{.directory}}
if [ "x{{.commit}}" != "x" ] && [ "x{{.commit}}" != "x{{.git.hash}}" ];  then
	cd {{.directory}}
	git reset --hard {{.commit}} || true
fi
EOF
    }]`)
	a, err := NewActionFromScript(b)
	assert.NotNil(t, a)
	assert.NoError(t, err)
	t.Logf("Action : %v", a)

	assert.Equal(t, len(a.Requirements), 2)
	assert.Equal(t, len(a.Parameters), 5)
	assert.Equal(t, len(a.Actions), 2)
}

func TestLoadFromRemoteActionScript(t *testing.T) {
	a, err := NewActionFromRemoteScript("https://raw.githubusercontent.com/ovh/cds/master/contrib/actions/cds-docker-package.hcl", nil)
	assert.NotNil(t, a)
	assert.NoError(t, err)
}

func TestTestLoadFromActionScriptWithJUnit(t *testing.T) {
	b := []byte(`
steps  = [{
	jUnitReport = "*.xml"
}]`)

	a, err := NewActionFromScript(b)
	assert.NotNil(t, a)
	assert.NoError(t, err)
	t.Logf("Action : %v", a)

	assert.Equal(t, JUnitAction, a.Actions[0].Name)
	assert.Equal(t, BuiltinAction, a.Actions[0].Type)
	assert.Equal(t, "path", a.Actions[0].Parameters[0].Name)
	assert.Equal(t, "*.xml", a.Actions[0].Parameters[0].Value)
}

func TestTestLoadFromActionScriptWithGitClone(t *testing.T) {
	b := []byte(`
steps  = [{
	GitClone = {
			directory = "./src"
			url = "{{.git.url}}"
			commit = "{{.git.hash}}"
			branch = "{{.git.branch}}"
	}
}]`)

	a, err := NewActionFromScript(b)
	assert.NotNil(t, a)
	assert.NoError(t, err)
	t.Logf("Action : %v", a)

	assert.Equal(t, GitCloneAction, a.Actions[0].Name)
	assert.Equal(t, BuiltinAction, a.Actions[0].Type)

	for i := 0; i < 4; i++ {
		name := a.Actions[0].Parameters[i].Name
		v := a.Actions[0].Parameters[i].Value
		switch name {
		case "directory":
			assert.Equal(t, "./src", v)
		case "url":
			assert.Equal(t, "{{.git.url}}", v)
		case "commit":
			assert.Equal(t, "{{.git.hash}}", v)
		case "branch":
			assert.Equal(t, "{{.git.branch}}", v)
		}
	}
}

func TestTestLoadFromActionScriptWithArtifactUpload(t *testing.T) {
	b := []byte(`
steps  = [{
	always_executed = true
	enabled = true
	artifactUpload = {
        path = "myartifact"
        tag = "{{.cds.version}}"
    }
}]`)

	a, err := NewActionFromScript(b)
	assert.NotNil(t, a)
	assert.NoError(t, err)
	t.Logf("Action : %v", a)

	assert.Equal(t, ArtifactUpload, a.Actions[0].Name)
	assert.Equal(t, BuiltinAction, a.Actions[0].Type)
	assert.Equal(t, true, a.Actions[0].AlwaysExecuted)
	assert.Equal(t, true, a.Actions[0].Enabled)
	var pathFound, tagFound bool
	for _, p := range a.Actions[0].Parameters {
		if p.Name == "path" {
			pathFound = true
			assert.Equal(t, "myartifact", p.Value)
		}

		if p.Name == "tag" {
			tagFound = true
			assert.Equal(t, "{{.cds.version}}", p.Value)
		}
	}

	assert.True(t, pathFound)
	assert.True(t, tagFound)
}

func TestTestLoadFromActionScriptWithArtifactDownload(t *testing.T) {
	b := []byte(`
steps  = [{
	artifactDownload = {
        path = "myartifact"
        tag = "{{.cds.version}}"
    }
}]`)

	a, err := NewActionFromScript(b)
	assert.NotNil(t, a)
	assert.NoError(t, err)
	t.Logf("Action : %v", a)

	assert.Equal(t, ArtifactDownload, a.Actions[0].Name)
	assert.Equal(t, BuiltinAction, a.Actions[0].Type)

	var pathFound, tagFound bool
	for _, p := range a.Actions[0].Parameters {
		if p.Name == "path" {
			pathFound = true
			assert.Equal(t, "myartifact", p.Value)
		}

		if p.Name == "tag" {
			tagFound = true
			assert.Equal(t, "{{.cds.version}}", p.Value)
		}
	}

	assert.True(t, pathFound)
	assert.True(t, tagFound)
}

func TestLoadFromActionScriptWithPlugin(t *testing.T) {
	b := []byte(`
steps  = [{
	plugin = {
        "my-plugin" = {
            "param1" = "value1"
            "param2" = "value2"
        }
    }
}]`)

	a, err := NewActionFromScript(b)
	assert.NotNil(t, a)
	assert.NoError(t, err)
	t.Logf("Action : %v", a)

	assert.Equal(t, "my-plugin", a.Actions[0].Name)
	assert.Equal(t, PluginAction, a.Actions[0].Type)

	// check param1
	param1Checked := false
	param2Checked := false
	for _, p := range a.Actions[0].Parameters {
		if p.Name == "param1" {
			assert.Equal(t, "value1", p.Value)
			param1Checked = true
		}
		if p.Name == "param2" {
			assert.Equal(t, "value2", p.Value)
			param2Checked = true
		}
	}

	assert.Equal(t, true, param1Checked)
	assert.Equal(t, true, param2Checked)

}

func TestTestLoadFromActionScriptWithError(t *testing.T) {
	b := []byte(`
steps  = [{
	blabla = "trololo"
}]`)

	a, err := NewActionFromScript(b)
	assert.Nil(t, a)
	assert.Error(t, err)
	t.Logf("Error : %s", err)
}

func TestTestDefautValues(t *testing.T) {
	b := []byte(`
	steps  = [{
	artifactDownload = {
				path = "myartifact"
				tag = "{{.cds.version}}"
		}
	},{
	always_executed = false
	enabled = false
	artifactDownload = {
				path = "myartifact"
				tag = "{{.cds.version}}"
		}
	},{
	always_executed = true
	enabled = true
	artifactDownload = {
				path = "myartifact"
				tag = "{{.cds.version}}"
		}
	}]`)

	a, err := NewActionFromScript(b)
	assert.NotNil(t, a)
	assert.NoError(t, err)
	t.Logf("Action : %v", a)

	assert.Equal(t, false, a.Actions[0].AlwaysExecuted)
	assert.Equal(t, true, a.Actions[0].Enabled)
	assert.Equal(t, false, a.Actions[1].AlwaysExecuted)
	assert.Equal(t, false, a.Actions[1].Enabled)
	assert.Equal(t, true, a.Actions[2].AlwaysExecuted)
	assert.Equal(t, true, a.Actions[2].Enabled)

}
