import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import * as vscode from 'vscode';

import { onContextChanged } from './events/context';
import { CDSWorkflowPreview } from "./preview";
import { Journal } from './utils/journal';
import { CDS } from './cds';
import { createContextStatusBarItem } from './components/context-status';
import { SetCurrentContext, registerCommand } from './commands';
import { updateContext } from './utils/context';
import { SetCurrentProjectCommand } from './commands/set-current-project';
import { updateProject } from './utils/project';
import { createProjectStatusBarItem } from './components/project-status';
import { Context } from './cds/models';
import { onGitRepositoryChanged, updateGitRepository } from './events/git-repository';
import { PreviewWorkflowCommand } from './commands/preview-workflow';
import { isCDSActionFile, isCDSWorkerModelFile, isCDSWorkflowFile } from './cds/file_utils';
import { ClearCacheCommand } from './commands/clear-cache';

const CDS_SCHEMA = 'cds';

const schemas = {
    'action': [
        new RegExp(/\/.cds\/actions\/.+\.ya?ml$/),
    ],
    'job': [
        new RegExp(/\/.cds\/jobs\/.+\.ya?ml$/),
    ],
    'workflow': [
        new RegExp(/\/.cds\/workflows\/.+\.ya?ml$/),
    ],
    'worker-model': [
        new RegExp(/\/.cds\/worker-models\/.+\.ya?ml$/),
    ],
    'workflow-template': [
        new RegExp(/\/.cds\/workflow-templates\/.+\.ya?ml$/),
    ],
};

export async function activate(context: vscode.ExtensionContext) {
    const yamlExtension = vscode.extensions.getExtension('redhat.vscode-yaml');
    if (!yamlExtension) {
        vscode.window.showErrorMessage(
            'The "YAML Language Support by Red Hat" extension is required for the CDS extension to work properly. Please install it and reload the window.'
        );
        return;
    }
    const yamlExtensionAPI = await yamlExtension.activate();

    // instanciate the preview component
    const workflowPreview = new CDSWorkflowPreview(context);

    // register the commands
    registerCommand(context, new ClearCacheCommand());
    registerCommand(context, new SetCurrentContext());
    registerCommand(context, new SetCurrentProjectCommand());
    registerCommand(context, new PreviewWorkflowCommand(workflowPreview));

    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(event => {
        if (event.affectsConfiguration('cds.config')) {
            updateContext();
        }
    }));

    context.subscriptions.push(vscode.window.onDidChangeActiveTextEditor(editor => {
        updateGitRepository();
        updateVscodeContext(editor);
    }));


    // when the CDS context has changed
    context.subscriptions.push(onContextChanged(async (context) => {
        if (context) {
            updateSchema(context);
        }
    }));

    // when the current git repository has changed
    context.subscriptions.push(onGitRepositoryChanged(async (repository) => {
        updateProject(repository);
    }));

    // register the schema provider
    yamlExtensionAPI.registerContributor(CDS_SCHEMA, onRequestSchemaURI, onRequestSchemaContent);

    // creates the status bar displaying the current context
    createContextStatusBarItem(context);

    // creates the status bar displaying the current project
    createProjectStatusBarItem(context);

    // init the update of the context
    updateContext();

    // init the update of the repository
    updateGitRepository();

    // init the vscode context
    updateVscodeContext(vscode.window.activeTextEditor);
}

async function updateSchema(context: Context) {
    Journal.logInfo(`Downloading schema for "${context.context}"...`);
    await CDS.downloadSchemas();
    Journal.logInfo(`Downloaded schema for "${context.context}"`);
}

function onRequestSchemaURI(resource: string): string | undefined {
    for (const [type, patterns] of Object.entries(schemas)) {
        for (const pattern of patterns) {
            if (pattern.test(resource)) {
                return `${CDS_SCHEMA}:${type}`;
            }
        }
    }
    return undefined;
}

function onRequestSchemaContent(schemaUri: string): string | undefined {
    const parsedUri = vscode.Uri.parse(schemaUri);

    if (parsedUri.scheme !== CDS_SCHEMA) {
        return undefined;
    }

    return getSchemaContent(parsedUri.path);
}

function getSchemaContent(name: string) {
    const schemaPath = path.join(os.homedir(), '.cds-schema', name + '.v2.schema.json');
    return fs.readFileSync(schemaPath, 'utf-8');
}

function updateVscodeContext(editor?: vscode.TextEditor) {
    // set vscode context
    if (editor?.document && isCDSWorkflowFile(editor.document)) {
        vscode.commands.executeCommand('setContext', 'isCDSWorkflowFile', true);
    } else {
        vscode.commands.executeCommand('setContext', 'isCDSWorkflowFile', false);
    }

    if (editor?.document && isCDSActionFile(editor.document)) {
        vscode.commands.executeCommand('setContext', 'isCDSActionFile', true);
    } else {
        vscode.commands.executeCommand('setContext', 'isCDSActionFile', false);
    }

    if (editor?.document && isCDSWorkerModelFile(editor.document)) {
        vscode.commands.executeCommand('setContext', 'isCDSWorkerModelFile', true);
    } else {
        vscode.commands.executeCommand('setContext', 'isCDSWorkerModelFile', false);
    }
}
