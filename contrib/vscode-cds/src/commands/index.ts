import * as vscode from 'vscode';

export interface Command {
    getID(): string
    run(): Promise<void>
}

export function registerCommand(context: vscode.ExtensionContext, command: Command) {
    context.subscriptions.push(vscode.commands.registerCommand(command.getID(), async () => {
        await command.run();
    }));
}

export { SetCurrentContextCommand as SetCurrentContext, SetCurrentContextCommandID } from './set-current-context';
export { SetCurrentProjectCommand as SetCurrentProject, SetCurrentProjectCommandID } from './set-current-project';
