package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromActionScript(t *testing.T) {
	b := []byte(`
name = "CDS_GitClone"
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
	a, err := NewActionFromRemoteScript("https://raw.githubusercontent.com/ovh/cds-contrib/master/actions/cds-git-clone.hcl", nil)
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

func TestTestLoadFromActionScriptWithArtifactUpload(t *testing.T) {
	b := []byte(`
steps  = [{
	final = true
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
	assert.Equal(t, true, a.Actions[0].Final)
	assert.Equal(t, true, a.Actions[0].Enabled)
	assert.Equal(t, "path", a.Actions[0].Parameters[0].Name)
	assert.Equal(t, "myartifact", a.Actions[0].Parameters[0].Value)
	assert.Equal(t, "tag", a.Actions[0].Parameters[1].Name)
	assert.Equal(t, "{{.cds.version}}", a.Actions[0].Parameters[1].Value)
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
	assert.Equal(t, "path", a.Actions[0].Parameters[0].Name)
	assert.Equal(t, "myartifact", a.Actions[0].Parameters[0].Value)
	assert.Equal(t, "tag", a.Actions[0].Parameters[1].Name)
	assert.Equal(t, "{{.cds.version}}", a.Actions[0].Parameters[1].Value)
}

func TestTestLoadFromActionScriptWithPlugin(t *testing.T) {
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
	assert.Equal(t, "param1", a.Actions[0].Parameters[0].Name)
	assert.Equal(t, "value1", a.Actions[0].Parameters[0].Value)
	assert.Equal(t, "param2", a.Actions[0].Parameters[1].Name)
	assert.Equal(t, "value2", a.Actions[0].Parameters[1].Value)

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
