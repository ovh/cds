import * as vscode from "vscode";
import * as uri from "vscode-uri";

export interface valid {
    valid: boolean;
    error: string;
}

export function isCDSWorkflowFile(document?: vscode.TextDocument): valid {
    const valid = isCDSFile(document);
    if (!valid.valid) {
        return valid;
    }
    if (getParentDirectory(document?.uri) !== 'workflows') {
        return {valid: false, error: `Workflow files must be inside the 'workflows' folder`};
    }
    return {valid: true, error: ''};
}

export function isCDSWorkerModelFile(document?: vscode.TextDocument): valid {
    const valid = isCDSFile(document);
    if (!valid.valid) {
        return valid;
    }

    if (getParentDirectory(document?.uri) !== 'worker-models') {
        return {valid: false, error: `Worker model files must be inside the 'worker-models' folder`};
    }
    return {valid: true, error: ''};
}

export function isCDSActionFile(document?: vscode.TextDocument): valid {
    const valid = isCDSFile(document);
    if (!valid.valid) {
        return valid;
    }

    if (getParentDirectory(document?.uri) !== 'actions') {
        return {valid: false, error: `Action files must be inside the 'actions' folder`};
    }
    return {valid: true, error: ''};
}

function isCDSFile(document?: vscode.TextDocument): valid { 
    if (!document) {
        return {valid: false, error: `Unable to get file`};
    }
    if (document.languageId !== 'yaml') {
        return {valid: false, error: `It's not a yaml file`};
    }
    if (document.isUntitled) {
        return {valid: false, error: `Unable to preview an untitled file`};
    }
    return {valid: true, error:''};
}

function getParentDirectory(filepath?: uri.URI): string {
    if (!filepath) {
        return '';
    }
    const dirPath = uri.Utils.dirname(filepath);
    const dirSplit = dirPath.toString().split('/');
    return dirSplit[dirSplit.length -1];
}