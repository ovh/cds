import * as vscode from 'vscode';

import { Journal } from './lib/utils/journal';
import { CDS } from './lib/cds';
import { selectContext } from './select-context';
import { init as initPreview } from "./preview";

let currentContextBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
    const setCurrentContextCommandID = 'vscode-cds.setCurrentContext';
    context.subscriptions.push(vscode.commands.registerCommand(setCurrentContextCommandID, async () => {
        await switchContext();
    }));

    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(event => {
        let affected = event.affectsConfiguration("cds.config");
        if (affected) {
            updateStatusBar();
        }
    }));

    currentContextBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    currentContextBarItem.command = setCurrentContextCommandID;
    currentContextBarItem.tooltip = 'Current CDS context';
    context.subscriptions.push(currentContextBarItem);

    CDS.getAvailableContexts();

    updateStatusBar();

    Journal.logInfo('Init CDS preview component');
    initPreview(context);
}

async function updateStatusBar(): Promise<void> {
    try {
        const context = await CDS.getCurrentContext();

        if (context) {
            currentContextBarItem.text = context.context;
            currentContextBarItem.show();
        } else {
            currentContextBarItem.hide();
        }
    } catch (e) {
        Journal.logError(new Error(`Cannot get the current context: ${e}`));
        currentContextBarItem.hide();
    }
}

async function switchContext(): Promise<void> {
    const context = await selectContext();
    try {
        await CDS.setCurrentContext(context.context);
        await updateStatusBar();
    } catch (e) {
        Journal.logError(e as Error);
    }
}
