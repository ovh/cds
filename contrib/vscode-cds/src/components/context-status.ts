import * as vscode from 'vscode';

import { onContextChanged } from '../events/context';
import { SetCurrentContextCommandID } from '../commands';

let instance: vscode.StatusBarItem;

export function createContextStatusBarItem(context: vscode.ExtensionContext) {
    instance = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    instance.command = SetCurrentContextCommandID;
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
