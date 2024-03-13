import * as vscode from "vscode";

import { Command } from ".";
import { isCDSWorkflowFile } from "../cds/file_utils";
import { CDSPreview } from "../preview";

export const PreviewWorkflowCommandID = 'vscode-cds.previewWorkflow';

export class PreviewWorkflowCommand implements Command {
    constructor(private instance: CDSPreview) { }

    getID(): string {
        return PreviewWorkflowCommandID
    }

    async run(): Promise<void> {
        if (vscode.window.activeTextEditor?.document.uri && isCDSWorkflowFile(vscode.window.activeTextEditor.document)) {
            this.instance.load(vscode.window.activeTextEditor?.document.uri);
        }
    }
}
