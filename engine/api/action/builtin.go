package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CreateBuiltinActions add builtin actions in database if needed
func CreateBuiltinActions(db *gorp.DbMap) error {
	// ----------------------------------- Script ---------------------------
	script := sdk.NewAction(sdk.ScriptAction)
	script.Type = sdk.BuiltinAction
	script.Description = `CDS Builtin Action.
Execute a script, written in script attribute.`
	script.Parameter(sdk.Parameter{
		Name: "script",
		Description: `Content of your script.
You can put #!/bin/bash, or #!/bin/perl at first line.
Make sure that the binary used is in
the pre-requisites of action`,
		Type: sdk.TextParameter})
	if err := checkBuiltinAction(db, script); err != nil {
		return err
	}

	// ----------------------------------- JUnit    ---------------------------
	junit := sdk.NewAction(sdk.JUnitAction)
	junit.Type = sdk.BuiltinAction
	junit.Description = `CDS Builtin Action.
Parse given file to extract Unit Test results.`
	junit.Parameter(sdk.Parameter{
		Name:        "path",
		Description: `Path to junit xml file.`,
		Type:        sdk.TextParameter})
	if err := checkBuiltinAction(db, junit); err != nil {
		return err
	}

	// ----------------------------------- Coverage ---------------------------
	cover := sdk.NewAction(sdk.CoverageAction)
	cover.Type = sdk.BuiltinAction
	cover.Description = `CDS Builtin Action.
Parse given file to extract coverage results.`
	cover.Parameter(sdk.Parameter{
		Name:        "format",
		Description: `Coverage report format.`,
		Type:        sdk.ListParameter,
		Value:       "lcov;cobertura",
	})
	cover.Parameter(sdk.Parameter{
		Name:        "path",
		Description: `Path of the coverage report file.`,
		Type:        sdk.StringParameter,
	})
	cover.Parameter(sdk.Parameter{
		Name:        "minimum",
		Description: `Minimum percentage of coverage required (-1 means no minimum).`,
		Type:        sdk.NumberParameter,
		Advanced:    true,
	})
	if err := checkBuiltinAction(db, cover); err != nil {
		return err
	}

	// ----------------------------------- Git clone    -----------------------
	gitclone := sdk.NewAction(sdk.GitCloneAction)
	gitclone.Type = sdk.BuiltinAction
	gitclone.Description = `CDS Builtin Action.
Clone a repository into a new directory.`

	gitclone.Parameter(sdk.Parameter{
		Name: "url",
		Description: `URL must contain information about the transport protocol, the address of the remote server, and the path to the repository.
If your application is linked to a repository, you can use {{.git.url}} (clone over ssh) or {{.git.http_url}} (clone over https)`,
		Value: "{{.git.url}}",
		Type:  sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:  "privateKey",
		Value: "",
		Description: `Set the private key to be able to git clone from ssh.
You can create an application key named 'app-key' and use it in this action.
The public key have to be granted on your repository`,
		Type: sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "user",
		Description: "Set the user to be able to git clone from https with authentication",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "password",
		Description: "Set the password to be able to git clone from https with authentication",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "branch",
		Description: "Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to {{.git.branch}} branch instead.",
		Value:       "{{.git.branch}}",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "commit",
		Description: "Set the current branch head (HEAD) to the commit.",
		Value:       "{{.git.hash}}",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "directory",
		Description: "The name of a directory to clone into.",
		Value:       "{{.cds.workspace}}",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "depth",
		Description: "gitClone use a depth of 50 by default. You can remove --depth with the value 'false'",
		Value:       "",
		Type:        sdk.StringParameter,
		Advanced:    true,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "submodules",
		Description: "gitClone clones submodules by default, you can set 'false' to avoid this",
		Value:       "true",
		Type:        sdk.BooleanParameter,
		Advanced:    true,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "tag",
		Description: "Useful when you want to git clone a specific tag",
		Value:       sdk.DefaultGitCloneParameterTagValue,
		Type:        sdk.StringParameter,
		Advanced:    true,
	})
	gitclone.Requirement("git", sdk.BinaryRequirement, "git")

	if err := checkBuiltinAction(db, gitclone); err != nil {
		return err
	}

	// ----------------------------------- Checkout Application    -----------------------
	checkoutApplication := sdk.NewAction(sdk.CheckoutApplicationAction)
	checkoutApplication.Type = sdk.BuiltinAction
	checkoutApplication.Description = `CDS Builtin Action.
Checkout a repository into a new directory.

This action use the configuration from application to git clone the repository.
The clone will be done with a depth of 50 and with submodules.
If you want to modify theses options, you have to use gitClone action.
`

	checkoutApplication.Parameter(sdk.Parameter{
		Name:        "directory",
		Description: "The name of a directory to clone into.",
		Value:       "{{.cds.workspace}}",
		Type:        sdk.StringParameter,
	})
	checkoutApplication.Requirement("git", sdk.BinaryRequirement, "git")

	if err := checkBuiltinAction(db, checkoutApplication); err != nil {
		return err
	}

	// ----------------------------------- Deploy Application    -----------------------
	deployApplication := sdk.NewAction(sdk.DeployApplicationAction)
	deployApplication.Type = sdk.BuiltinAction
	deployApplication.Description = `CDS Builtin Action.
Deploy an application of a integration.`

	if err := checkBuiltinAction(db, deployApplication); err != nil {
		return err
	}

	// ----------------------------------- Git tag    -----------------------
	gittag := sdk.NewAction(sdk.GitTagAction)
	gittag.Type = sdk.BuiltinAction
	gittag.Description = `CDS Builtin Action.
Tag the current branch and push it.
Semver used if fully compatible with https://semver.org/
`

	gittag.Parameter(sdk.Parameter{
		Name:        "tagPrerelease",
		Description: "Prerelase version of the tag. Example: alpha on a tag 1.0.0 will return 1.0.0-apha",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "tagLevel",
		Description: "Set the level of the tag. Must be 'major' or 'minor' or 'patch'",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "tagMetadata",
		Description: "Metadata of the tag. Example: cds.42 on a tag 1.0.0 will return 1.0.0+cds.42",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "tagMessage",
		Description: "Set a message for the tag.",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "path",
		Description: "The path to your git directory.",
		Value:       "{{.cds.workspace}}",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "prefix",
		Description: "Prefix for tag name",
		Value:       "",
		Type:        sdk.StringParameter,
		Advanced:    true,
	})
	gittag.Requirement("git", sdk.BinaryRequirement, "git")
	gittag.Requirement("gpg", sdk.BinaryRequirement, "gpg")

	if err := checkBuiltinAction(db, gittag); err != nil {
		return err
	}

	// ----------------------------------- Git Release -----------------------
	gitrelease := sdk.NewAction(sdk.ReleaseAction)
	gitrelease.Type = sdk.BuiltinAction
	gitrelease.Description = `CDS Builtin Action. Make a release using repository manager.`

	gitrelease.Parameter(sdk.Parameter{
		Name:        "tag",
		Description: "Tag name.",
		Value:       "{{.cds.release.version}}",
		Type:        sdk.StringParameter,
	})
	gitrelease.Parameter(sdk.Parameter{
		Name:        "title",
		Value:       "",
		Description: "Set a title for the release",
		Type:        sdk.StringParameter,
	})
	gitrelease.Parameter(sdk.Parameter{
		Name:        "releaseNote",
		Description: "Set a release note for the release",
		Type:        sdk.TextParameter,
	})
	gitrelease.Parameter(sdk.Parameter{
		Name:        "artifacts",
		Description: "Set a list of artifacts, separate by , . You can also use regexp.",
		Type:        sdk.StringParameter,
	})
	if err := checkBuiltinAction(db, gitrelease); err != nil {
		return err
	}

	// ----------------------------------- Serve Static Files -----------------------
	serveStaticAct := sdk.NewAction(sdk.ServeStaticFiles)
	serveStaticAct.Type = sdk.BuiltinAction
	serveStaticAct.Description = `CDS Builtin Action
	Useful to upload static files and serve them.
	For example your report about coverage, tests, performances, ...`
	serveStaticAct.Parameter(sdk.Parameter{
		Name:        "name",
		Description: "Name to display in CDS UI and identify your static files",
		Type:        sdk.StringParameter})
	serveStaticAct.Parameter(sdk.Parameter{
		Name:        "path",
		Description: "Path where static files will be uploaded (example: mywebsite/*). If it's a file, the entrypoint would be set to this filename by default.",
		Type:        sdk.StringParameter})
	serveStaticAct.Parameter(sdk.Parameter{
		Name:        "entrypoint",
		Description: "Filename (and not path) for the entrypoint when serving static files (default: if empty it would be index.html)",
		Type:        sdk.StringParameter,
		Value:       "",
		Advanced:    true})
	serveStaticAct.Parameter(sdk.Parameter{
		Name:        "static-key",
		Description: "Indicate a static-key which will be a reference to keep the same generated URL. Example: {{.git.branch}}",
		Type:        sdk.StringParameter,
		Value:       "",
		Advanced:    true})
	serveStaticAct.Parameter(sdk.Parameter{
		Name:        "destination",
		Description: "Destination of uploading. Use the name of integration attached on your project",
		Value:       "", // empty is the default value
		Type:        sdk.StringParameter,
		Advanced:    true,
	})

	if err := checkBuiltinAction(db, serveStaticAct); err != nil {
		return err
	}

	artifactUpload := craftBuiltinArtifactUploadAction()
	if err := checkBuiltinAction(db, artifactUpload); err != nil {
		return err
	}

	artifactDownload := craftBuiltinArtifactDownloadAction()
	if err := checkBuiltinAction(db, artifactDownload); err != nil {
		return err
	}

	return nil
}

func craftBuiltinArtifactUploadAction() *sdk.Action {
	upload := sdk.NewAction(sdk.ArtifactUpload)
	upload.Type = sdk.BuiltinAction
	upload.Parameter(sdk.Parameter{
		Name:        "path",
		Type:        sdk.StringParameter,
		Description: "Path of file to upload, example: ./src/yourFile.json",
	})
	upload.Parameter(sdk.Parameter{
		Name:        "tag",
		Type:        sdk.StringParameter,
		Description: "Artifact will be uploaded with a tag, generally {{.cds.version}}",
		Value:       "{{.cds.version}}",
	})
	upload.Parameter(sdk.Parameter{
		Name:        "enabled",
		Type:        sdk.BooleanParameter,
		Description: "Enable artifact upload",
		Value:       "true",
		Advanced:    true,
	})
	upload.Parameter(sdk.Parameter{
		Name:        "destination",
		Description: "Destination of this artifact. Use the name of integration attached on your project",
		Value:       "", // empty is the default value
		Type:        sdk.StringParameter,
		Advanced:    true,
	})
	return upload
}

func craftBuiltinArtifactDownloadAction() *sdk.Action {
	dl := sdk.NewAction(sdk.ArtifactDownload)
	dl.Type = sdk.BuiltinAction
	dl.Parameter(sdk.Parameter{
		Name:        "path",
		Description: "Path where artifacts will be downloaded",
		Type:        sdk.StringParameter,
	})
	dl.Parameter(sdk.Parameter{
		Name:        "tag",
		Description: "Artifact are uploaded with a tag, generally {{.cds.version}}",
		Type:        sdk.StringParameter,
		Value:       "{{.cds.version}}",
	})
	dl.Parameter(sdk.Parameter{
		Name:        "enabled",
		Type:        sdk.BooleanParameter,
		Description: "Enable artifact download",
		Value:       "true",
		Advanced:    true,
	})
	dl.Parameter(sdk.Parameter{
		Name:        "pattern",
		Type:        sdk.StringParameter,
		Description: "Empty: download all files. Otherwise, enter regexp pattern to choose file: (fileA|fileB)",
		Value:       "",
		Advanced:    true,
	})
	return dl
}

// checkBuiltinAction add builtin actions in database if needed
func checkBuiltinAction(db *gorp.DbMap, a *sdk.Action) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()

	nb, err := tx.SelectInt("SELECT COUNT(1) FROM action WHERE action.name = $1 and action.type = $2", a.Name, sdk.BuiltinAction)
	if err != nil {
		return sdk.WrapError(err, "unable to count action %s", a.Name)
	}

	// If the action doesn't exist, let's create
	if nb == 0 {
		log.Debug("createBuiltinAction> create builtin action %s", a.Name)
		if err := Insert(tx, a); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}

	id, err := tx.SelectInt("SELECT id FROM action WHERE action.name = $1 and action.type = $2", a.Name, sdk.BuiltinAction)
	if err != nil {
		return sdk.WrapError(err, "unable to get action %s ID", a.Name)
	}

	a.ID = id
	log.Debug("createBuiltinAction> update builtin action %s", a.Name)
	if err := Update(tx, a); err != nil {
		return err
	}
	return sdk.WithStack(tx.Commit())
}
