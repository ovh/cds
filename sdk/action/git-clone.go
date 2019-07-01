package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// GitClone action definition.
var GitClone = Manifest{
	Action: sdk.Action{
		Name:        sdk.GitCloneAction,
		Description: "Clone a repository into a new directory.",
		Parameters: []sdk.Parameter{
			{
				Name: "url",
				Description: `URL must contain information about the transport protocol, the address of the remote server, and the path to the repository.
If your application is linked to a repository, you can use {{.git.url}} (clone over ssh) or {{.git.http_url}} (clone over https).`,
				Value: "{{.git.url}}",
				Type:  sdk.StringParameter,
			},
			{
				Name:  "privateKey",
				Value: "",
				Description: `(optional) Set the private key to be able to git clone from ssh.
You can create an application key named 'app-key' and use it in this action.
The public key have to be granted on your repository.`,
				Type: sdk.KeySSHParameter,
			},
			{
				Name:        "user",
				Description: "(optional) Set the user to be able to git clone from https with authentication.",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "password",
				Description: "(optional) Set the password to be able to git clone from https with authentication.",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "branch",
				Description: "(optional) Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to {{.git.branch}} branch instead.",
				Value:       "{{.git.branch}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "commit",
				Description: "(optional) Set the current branch head (HEAD) to the commit.",
				Value:       "{{.git.hash}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "directory",
				Description: "(optional) The name of a directory to clone into.",
				Value:       "{{.cds.workspace}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "depth",
				Description: "(optional) Clone with a depth of 50 by default. You can remove --depth with the value 'false'.",
				Value:       "",
				Type:        sdk.StringParameter,
				Advanced:    true,
			},
			{
				Name:        "submodules",
				Description: "(optional) Submodules are cloned by default, you can set 'false' to avoid this.",
				Value:       "true",
				Type:        sdk.BooleanParameter,
				Advanced:    true,
			},
			{
				Name:        "tag",
				Description: "(optional) Useful when you want to git clone a specific tag. Empty by default, you can set to `{{.git.tag}}` to clone a tag from your repository. In this way, in your workflow payload you can add a key in your JSON like \"git.tag\": \"1.0.2\".",
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
	Example: exportentities.PipelineV1{
		Version: exportentities.PipelineVersion1,
		Name:    "Pipeline1",
		Stages:  []string{"Stage1"},
		Jobs: []exportentities.Job{{
			Name:  "Job1",
			Stage: "Stage1",
			Steps: []exportentities.Step{
				{
					GitClone: &exportentities.StepGitClone{
						URL:        "{{.git.url}}",
						Branch:     "{{.git.branch}}",
						Commit:     "{{.git.commit}}",
						PrivateKey: "proj-mykey",
						Directory:  "{{.cds.workspace}}",
					},
				},
			},
		}},
	},
}
