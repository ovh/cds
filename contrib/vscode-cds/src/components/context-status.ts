import * as vscode from 'vscode';

import { setCurrentContextCommandID } from '../commands/set-current-context';
import { onContextChanged } from '../events/context';

let instance: vscode.StatusBarItem;

export function createContextStatusBarItem(context: vscode.ExtensionContext) {
    instance = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    instance.command = setCurrentContextCommandID;
    instance.tooltip = 'Current CDS context';
    context.subscriptions.push(instance);

    context.subscriptions.push(onContextChanged(context => {
        if (context) {
            instance.text = context.context;
            instance.show();
        } else {
            instance.hide();
        }
    }));
}
