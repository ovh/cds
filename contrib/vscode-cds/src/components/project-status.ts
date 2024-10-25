import * as vscode from 'vscode';

import { SetCurrentProjectCommandID } from '../commands';
import { onProjectChanged } from '../events/project';
import { onGitRepositoryChanged } from '../events/git-repository';

let instance: vscode.StatusBarItem;

export function createProjectStatusBarItem(context: vscode.ExtensionContext) {
    instance = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    instance.command = SetCurrentProjectCommandID;
    instance.tooltip = 'Current CDS project';
    context.subscriptions.push(instance);

    context.subscriptions.push(onGitRepositoryChanged(repository => {
        if (repository) {
            instance.show();
        } else {
            instance.hide();
        }
    }));

    context.subscriptions.push(onProjectChanged(project => {
        if (project) {
            instance.text = `${project.name} (${project.key})`;
        } else {
            instance.text = 'Select a CDS project';
        }

        instance.color = (!!!project || project.found) ? '' : '#FF0000';
        instance.show();
    }));
}
