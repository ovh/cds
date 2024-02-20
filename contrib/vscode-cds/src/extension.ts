import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import * as vscode from 'vscode';

import { Journal } from './lib/utils/journal';
import { CDS } from './lib/cds';
import { selectContext } from './forms/select-context';
import { onContextChanged, setContext } from './events/context';
import { init as initPreview } from "./preview";
import { createContextStatusBarItem } from './components/context-status';

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

    Journal.logInfo('Activating CDS Extension');

    const setCurrentContextCommandID = 'vscode-cds.setCurrentContext';
    context.subscriptions.push(vscode.commands.registerCommand(setCurrentContextCommandID, async () => {
        await switchContext();
    }));

    initPreview(context);

    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(event => {
        if (event.affectsConfiguration('cds.config')) {
            updateContext();
        }
    }));

    context.subscriptions.push(onContextChanged(async (context) => {
        if (context) {
            Journal.logInfo(`Downloading schema for "${context.context}"...`);
            await CDS.downloadSchemas();
            Journal.logInfo(`Downloaded schema for "${context.context}"`);
        }
    }));

    // register the schema provider
    yamlExtensionAPI.registerContributor(CDS_SCHEMA, onRequestSchemaURI, onRequestSchemaContent);

    // creates the status bar displaying the current context
    createContextStatusBarItem(context);

    // init the update of the context
    updateContext();
}

async function updateContext(): Promise<void> {
    try {
        const context = await CDS.getCurrentContext();
        setContext(context);
    } catch (e) {
        Journal.logError(new Error(`Cannot get the current context: ${e}`));
        setContext(null);
    }
}

async function switchContext(): Promise<void> {
    const context = await selectContext();
    try {
        await CDS.setCurrentContext(context.context);
        await updateContext();
    } catch (e) {
        Journal.logError(e as Error);
    }
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
