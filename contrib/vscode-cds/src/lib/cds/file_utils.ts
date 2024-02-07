import * as vscode from "vscode";
import * as uri from "vscode-uri";
import { Journal } from "../utils/journal";

export function isCDSWorkflowFile(document?: vscode.TextDocument): boolean {
    if (!isCDSFile(document)) {
        return false;
    }

    if (getParentDirectory(document?.uri) !== 'workflows') {
        Journal.logError(new Error(`Workflow files must be inside the 'workflows' folder`));
        return false;
    }
    Journal.logInfo('CDS workflow file detected');
    return true;
}

export function isCDSWorkerModelFile(document?: vscode.TextDocument): boolean {
    if (!isCDSFile(document)) {
        return false;
    }

    if (getParentDirectory(document?.uri) !== 'worker-models') {
        Journal.logError(new Error(`Workflow files must be inside the 'worker-models' folder`));
        return false;
    }
    return true;
}

export function isCDSActionFile(document?: vscode.TextDocument): boolean {
    if (!isCDSFile(document)) {
        return false;
    }

    if (getParentDirectory(document?.uri) !== 'actions') {
        Journal.logError(new Error(`Workflow files must be inside the 'actions' folder`));
        return false;
    }
    return true;
}

function isCDSFile(document?: vscode.TextDocument): boolean { 
    if (!document) {
        Journal.logError(new Error(`Unable to get file`));
        return false;
    }
    if (document.languageId !== 'yaml') {
        Journal.logError(new Error(`It's not a yaml file`));
        return false;
    }
    if (document.isUntitled) {
        Journal.logError(new Error(`Unable to preview an untitled file`));
        return false;
    }
    return true;
}

function getParentDirectory(filepath?: uri.URI): string {
    if (!filepath) {
        return '';
    }
    const dirPath = uri.Utils.dirname(filepath);
    const dirSplit = dirPath.toString().split('/');
    return dirSplit[dirSplit.length -1];
}