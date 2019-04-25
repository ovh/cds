package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// GitClone action definition.
var GitClone = Manifest{
	Action: sdk.Action{
		Name: sdk.GitCloneAction,
		Description: `CDS Builtin Action.
Clone a repository into a new directory.`,
		Parameters: []sdk.Parameter{
			{
				Name: "url",
				Description: `URL must contain information about the transport protocol, the address of the remote server, and the path to the repository.
If your application is linked to a repository, you can use {{.git.url}} (clone over ssh) or {{.git.http_url}} (clone over https)`,
				Value: "{{.git.url}}",
				Type:  sdk.StringParameter,
			},
			{
				Name:  "privateKey",
				Value: "",
				Description: `Set the private key to be able to git clone from ssh.
You can create an application key named 'app-key' and use it in this action.
The public key have to be granted on your repository`,
				Type: sdk.StringParameter,
			},
			{
				Name:        "user",
				Description: "Set the user to be able to git clone from https with authentication",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "password",
				Description: "Set the password to be able to git clone from https with authentication",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "branch",
				Description: "Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to {{.git.branch}} branch instead.",
				Value:       "{{.git.branch}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "commit",
				Description: "Set the current branch head (HEAD) to the commit.",
				Value:       "{{.git.hash}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "directory",
				Description: "The name of a directory to clone into.",
				Value:       "{{.cds.workspace}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "depth",
				Description: "gitClone use a depth of 50 by default. You can remove --depth with the value 'false'",
				Value:       "",
				Type:        sdk.StringParameter,
				Advanced:    true,
			},
			{
				Name:        "submodules",
				Description: "gitClone clones submodules by default, you can set 'false' to avoid this",
				Value:       "true",
				Type:        sdk.BooleanParameter,
				Advanced:    true,
			},
			{
				Name:        "tag",
				Description: "Useful when you want to git clone a specific tag",
				Value:       sdk.DefaultGitCloneParameterTagValue,
				Type:        sdk.StringParameter,
				Advanced:    true,
			},
		},
		Requirements: []sdk.Requirement{
			sdk.Requirement{
				Name:  "git",
				Type:  sdk.BinaryRequirement,
				Value: "git",
			},
		},
	},
	Example: exportentities.Step{
		GitClone: &exportentities.StepGitClone{
			URL:        "{{.git.url}}",
			Branch:     "{{.git.branch}}",
			Commit:     "{{.git.commit}}",
			PrivateKey: "proj-mykey",
			Directory:  "{{.cds.workspace}}",
		},
	},
}
