# VSCode CDS Extension

> navigate through CDS directly in VS Code

This extension allows you browse and monitor your CDS Workflow in Visual Studio Code. The support includes:

- Direct access to your favorites Workflows and Projects.
- Browse through you CDS Project
- View CDS Queue Waiting & Building
- Direct access to current workflow linked to your current open git repository
- CDS Yaml Completion

# Getting Started

It's easy to get started with CDS for Visual Studio Code. Simply follow these steps to get started.

1. Install and run cdsctl command line outside VS Code.
1. Make sure you have VSCode version 1.42.0 or higher.
1. Download the extension from [the marketplace](https://marketplace.visualstudio.com/vscode).
1. You should be good to go!


# Dev


``` bash
# build
$ git clone https://github.com/ovh/cds.git
$ cd cds/contrib/vscode-cds
$ npm install

# package
$ npm install -g vsce
# this will generate a vsix file, that you can manually install on your vscode
```

If you open the cds (https://github.com/ovh/cds.git) directory on your VSCode, you can launch the task `CDS Extension Run` (F5).