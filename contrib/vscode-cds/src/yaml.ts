import { workspace, Uri, ConfigurationTarget, window, Extension, extensions, commands } from "vscode";

export const VSCODE_YAML_EXTENSION_ID = 'redhat.vscode-yaml';

export const YAML_SCHEMA_CONFIG_NAME_OF_VSCODE_YAML_EXTENSION = "yaml.schemas";

export async function registerYamlSchemaSupport() {
    const config = workspace.getConfiguration().inspect(YAML_SCHEMA_CONFIG_NAME_OF_VSCODE_YAML_EXTENSION);

    let newValue: any = {};
    if (config) {
        newValue = Object.assign({}, config.globalValue);
    }

    // delete existing keys if already here
    Object.keys(newValue).forEach((key) => {
        if (key.indexOf("cds-schemas") > 0) {
            delete newValue[key];
        }
    });

    const homedir = require('os').homedir();
    newValue[Uri.file(`${homedir}/.cds-schema/workflow.schema.json`).toString()] = "*.cds*.yml";
    newValue[Uri.file(`${homedir}/.cds-schema/application.schema.json`).toString()] =  "*.cds*.app.yml";
    newValue[Uri.file(`${homedir}/.cds-schema/environment.schema.json`).toString()] =  "*.cds*.env.yml";
    newValue[Uri.file(`${homedir}/.cds-schema/pipeline.schema.json`).toString()] =  "*.cds*.pip.yml";

    await workspace.getConfiguration().update(YAML_SCHEMA_CONFIG_NAME_OF_VSCODE_YAML_EXTENSION, newValue, ConfigurationTarget.Global);
}

// Find redhat.vscode-yaml extension and try to activate it
export async function activateYamlExtension() {
    const ext: Extension<any> | undefined = extensions.getExtension(VSCODE_YAML_EXTENSION_ID);
    if (!ext) {
        window.showWarningMessage('Please install \'YAML Support by Red Hat\' via the Extensions pane.', 'install yaml extension')
            .then(() => {
                commands.executeCommand('workbench.extensions.installExtension', VSCODE_YAML_EXTENSION_ID);
            });
        return;
    }
    const yamlPlugin = await ext.activate();

    if (!yamlPlugin || !yamlPlugin.registerContributor) {
        window.showWarningMessage('The installed Red Hat YAML extension doesn\'t support Intellisense. Please upgrade \'YAML Support by Red Hat\' via the Extensions pane.');
        return;
    }
    return yamlPlugin;
}