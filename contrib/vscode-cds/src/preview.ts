import * as vscode from "vscode";
import * as uri from "vscode-uri";
import * as fs from 'fs';
import * as path from 'path';

import { isCDSWorkflowFile, isCDSWorkflowTemplateFile } from "./cds/file_utils";
import { Journal } from "./utils/journal";
import { Messenger} from "vscode-messenger";
import { GenerateWorkflow, GenerateWorkflowDataResponse, WorkflowRefresh, WorkflowTemplate, WorkflowTemplateGenerated } from "./type";
import { CDS } from "./cds";
import { WorkflowGenerateRequest } from "./cds/models/WorkflowGenerated";

const dirWeb = 'dist-web/workflow-preview';

export type RefreshMsg = {
    content: any;
    type: string;
}

export const Refresh = {
    method: 'refresh'
};

export class CDSWorkflowPreview extends vscode.Disposable {
    private static viewType = "cds.preview";

    private _panel?: vscode.WebviewPanel;
    private _resource?: vscode.Uri;
    private _resourceType?: string;
    private messenger: Messenger;
    private disposable: any;


    constructor(private _context: vscode.ExtensionContext) {
        super(() => {
            this.dispose();
        });
        this.messenger = new Messenger({debugLog: true});
        this.disposable = this.messenger.onRequest(GenerateWorkflow, async (request) => {
            Journal.logInfo('Calling CDS');
            if (this._resource?.path) {
                let req: WorkflowGenerateRequest = {filePath: this._resource?.path, params: request.parameters};
                const resp = await CDS.generateWorkflowFromTemplate(req);
                if (resp['workflow']) {
                    // Create file with workflow data
                    let filepath = process.env.TMPDIR + '/workflow-' + path.parse(this._resource?.path).base;
                    let ws = fs.createWriteStream(filepath);
                    ws.write(resp['workflow']);
                    ws.close();

                    vscode.workspace.openTextDocument(filepath).then(doc => {
                        vscode.window.showTextDocument(doc);
                    });
                }
                let r: GenerateWorkflowDataResponse = {workflow: resp.workflow, error: resp.error};
                this.messenger.sendNotification(WorkflowTemplateGenerated, {type: 'webview', webviewType: CDSWorkflowPreview.viewType}, { 'workflow': r.workflow});
                return r;
            }
        });
        
        
        _context.subscriptions.push(
            vscode.window.onDidChangeActiveTextEditor(editor => {
                if (this._panel && editor && isCDSWorkflowFile(editor.document)) {
                    this.load(editor.document.uri, 'workflow');
                }
                if (this._panel && editor && isCDSWorkflowTemplateFile(editor.document)) {
                    this.load(editor.document.uri, 'workflow-template');
                }
            })
        );

        _context.subscriptions.push(
            vscode.workspace.onDidSaveTextDocument(document => {
                if (document.uri === this._resource) {
                    this.refresh();
                }
            })
        );
    }

    public load(resource: vscode.Uri, type: string) {
        Journal.logInfo(`Loading preview of ${resource}`);

        this._resource = resource;
        this._resourceType = type;

        // Create panel webview
        if (!this._panel) {
            this._panel = vscode.window.createWebviewPanel(
                CDSWorkflowPreview.viewType,
                "CDS Workflow Preview",
                vscode.ViewColumn.Two,
                {
                    enableScripts: true,
                    localResourceRoots: [
                        vscode.Uri.joinPath(this._context.extensionUri, dirWeb),
                    ],
                    retainContextWhenHidden: true,
                }
            );
            this.messenger.registerWebviewPanel(this._panel);

            this._panel.onDidDispose(() => {
                if (this.disposable) {
                    this.disposable.dispose();
                }
                this._panel = undefined;
            });

            this._panel.webview.onDidReceiveMessage((msg: { type: string; value?: any }) => {
                switch (msg.type) {
                    case 'initialized':
                        this.refresh();
                        break;
                    default:
                        Journal.logError(new Error(`Unknown message type: ${msg.type}`));
                }
            });

            this._panel.webview.html = this.getHtmlContent();
        }

        // set the title
        this._panel.title = 'Preview ' + uri.Utils.basename(this._resource);

        // draw the preview
        this.refresh();
    }

    // Refresh the webview
    public refresh() {
        if (this._panel && this._resource && this._resourceType) {
            vscode.workspace.openTextDocument(this._resource).then(document => {
                if (this._resourceType === 'workflow') {
                    this.messenger.sendNotification(WorkflowRefresh, {type: 'webview', webviewType: CDSWorkflowPreview.viewType}, { 'workflow': document.getText()});
                } else if (this._resourceType === 'workflow-template') {
                    this.messenger.sendNotification(WorkflowTemplate, {type: 'webview', webviewType: CDSWorkflowPreview.viewType}, { 'workflowTemplate': document.getText()});
                }
            });
        }
    }

    private getHtmlContent() {
        if (!this._panel) {
            return '';
        }

        const stylesUri = this._panel.webview.asWebviewUri(
            vscode.Uri.joinPath(this._context.extensionUri, dirWeb, "styles.css")
        );

        const scriptPolyfillsUri = this._panel.webview.asWebviewUri(
            vscode.Uri.joinPath(this._context.extensionUri, dirWeb, "polyfills.js")
        );

        const scriptRuntimeUri = this._panel.webview.asWebviewUri(
            vscode.Uri.joinPath(this._context.extensionUri, dirWeb, "runtime.js")
        );

        const scriptMainUri = this._panel.webview.asWebviewUri(
            vscode.Uri.joinPath(this._context.extensionUri, dirWeb,
                "main.js")
        );

        const baseUri = this._panel.webview.asWebviewUri(vscode.Uri.joinPath(
            this._context.extensionUri, dirWeb)
        ).toString().replace('%22', '');

        return `
            <!doctype html>
            <html lang="en">
                <head>
                    <meta charset="utf-8">
                    <title>CDS.Preview</title>
                    <base href="${baseUri}/">
                    <meta name="viewport" content="width=device-width, initial-scale=1">
                    <link rel="stylesheet" href="${stylesUri}">
                </head>
                <body>
                    <app-root></app-root>
                    <script src="${scriptPolyfillsUri}" type="module"></script>
                    <script src="${scriptMainUri}" type="module"></script>
                    <script src="${scriptRuntimeUri}" type="module"></script>
                </body>
            </html>
        `;
    }
}
