import * as vscode from 'vscode';

import { Journal } from './lib/utils/journal';
import { CDS } from './lib/cds';
import { selectContext } from './forms/select-context';
import { onContextChanged, setContext } from './events/context';
import { init as initPreview } from "./preview";

let currentContextBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
    Journal.logInfo('Activating CDS Extension');

    const setCurrentContextCommandID = 'vscode-cds.setCurrentContext';
    context.subscriptions.push(vscode.commands.registerCommand(setCurrentContextCommandID, async () => {
        await switchContext();
    }));

    initPreview(context);

    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(event => {
        if (event.affectsConfiguration("cds.config")) {
            updateContext();
        }
    }));

    const contextChanged = onContextChanged(context => {
        Journal.logInfo(`CDS context has changed to "${context?.context ?? null}"`);

        if (context) {
            currentContextBarItem.text = context.context;
            currentContextBarItem.show();
        } else {
            currentContextBarItem.hide();
        }
    })
    context.subscriptions.push(contextChanged);

    currentContextBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    currentContextBarItem.command = setCurrentContextCommandID;
    currentContextBarItem.tooltip = 'Current CDS context';
    context.subscriptions.push(currentContextBarItem);

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
