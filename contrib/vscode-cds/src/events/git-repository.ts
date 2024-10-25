import * as vscode from 'vscode';

import { getGitRepositoryPath } from '../utils/git';
import { Journal } from '../utils/journal';

const emitter = new vscode.EventEmitter<string | null>();

let currentRepository: string | null = null;

export const onGitRepositoryChanged = emitter.event;

export function setGitRepository(path: string | null) {
    if (path !== currentRepository) {
        currentRepository = path;
        emitter.fire(path);
    }
}

export async function updateGitRepository() {
    // ignore when we're going to an output panel
    if (vscode.window.activeTextEditor?.document.uri.scheme === 'output') {
        return;
    }

    if (!vscode.window.activeTextEditor || vscode.window.activeTextEditor.document.uri.scheme !== 'file') {
        setGitRepository(null);
    } else {
        setGitRepository(await getGitRepositoryPath(vscode.window.activeTextEditor.document.uri.path))
    }
}
