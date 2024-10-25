import * as vscode from "vscode";

import { Command } from ".";
import { isCDSWorkflowFile, isCDSWorkflowTemplateFile } from "../cds/file_utils";
import { CDSWorkflowPreview } from "../preview";
import { Journal } from "../utils/journal";

export const PreviewWorkflowCommandID = 'vscode-cds.previewWorkflow';

export class PreviewWorkflowCommand implements Command {
    constructor(private instance: CDSWorkflowPreview) { }

    getID(): string {
        return PreviewWorkflowCommandID;
    }

    async run(): Promise<void> {
        if (vscode.window.activeTextEditor?.document.uri && isCDSWorkflowFile(vscode.window.activeTextEditor.document)) {
            this.instance.load(vscode.window.activeTextEditor?.document.uri, 'workflow');
        }
        if (vscode.window.activeTextEditor?.document.uri && isCDSWorkflowTemplateFile(vscode.window.activeTextEditor.document)) {
            this.instance.load(vscode.window.activeTextEditor?.document.uri, 'workflow-template');
        }
    }
}
