import * as vscode from "vscode";
import * as uri from "vscode-uri";
import * as path from "path";

export function isCDSWorkflowTemplateFile(document: vscode.TextDocument) {
    if (!isCDSFile(document)) {
        return false;
    }

    if (getParentDirectory(document?.uri) !== 'workflow-templates') {
        return false;
    }

    return true;
}

export function isCDSWorkflowFile(document: vscode.TextDocument) {
    if (!isCDSFile(document)) {
        return false;
    }

    if (getParentDirectory(document?.uri) !== 'workflows') {
        return false;
    }

    return true;
}

export function isCDSWorkerModelFile(document: vscode.TextDocument) {
    if (!isCDSFile(document)) {
        return false;
    }

    if (getParentDirectory(document?.uri) !== 'worker-models') {
        return false;
    }

    return true;
}

export function isCDSActionFile(document: vscode.TextDocument) {
    if (!isCDSFile(document)) {
        return false;
    }

    if (getParentDirectory(document?.uri) !== 'actions') {
        return false;
    }

    return true;
}

function isCDSFile(document: vscode.TextDocument) {
    if (document.languageId !== 'yaml') {
        return false;
    }

    if (document.isUntitled) {
        return false;
    }

    return getParentDirectory(getParentPath(document.uri)) === '.cds';
}

function getParentPath(filepath: uri.URI) {
    return uri.Utils.dirname(uri.Utils.resolvePath(filepath));
}

function getParentDirectory(filepath: uri.URI): string {
    return path.basename(getParentPath(filepath).path);
}
