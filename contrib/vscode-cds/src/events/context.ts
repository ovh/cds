import * as vscode from 'vscode';
import { Context } from '../cds/models';

const emitter = new vscode.EventEmitter<Context | null>();

export const onContextChanged = emitter.event;

export function setContext(context: Context | null) {
    emitter.fire(context);
}
