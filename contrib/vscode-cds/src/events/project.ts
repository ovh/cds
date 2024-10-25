import * as vscode from 'vscode';

import { Project } from '../cds/models';

const emitter = new vscode.EventEmitter<Project | null>();

export const onProjectChanged = emitter.event;

export function setProject(project: Project | null) {
    emitter.fire(project);
}
