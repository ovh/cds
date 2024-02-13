import * as vscode from "vscode";
import * as uri from "vscode-uri";
import { isCDSWorkflowFile } from "./lib/cds/file_utils";
import { Journal } from "./lib/utils/journal";

const dirWeb = 'dist-web';

export function init(context: vscode.ExtensionContext) {
    const cdsPreview = new CDSPreview(context);

    context.subscriptions.push(vscode.commands.registerCommand("vscode-cds.previewWorkflow", () => {
        Journal.logInfo('Running workflow preview on ' + vscode.window.activeTextEditor?.document.fileName);
        if ( isCDSWorkflowFile(vscode.window.activeTextEditor?.document) && vscode.window.activeTextEditor?.document.uri) {
            cdsPreview.load(vscode.window.activeTextEditor?.document.uri);
          }
        })
    );
}

class CDSPreview extends vscode.Disposable {
    private static viewType = "cds.preview";

    private _panel?: vscode.WebviewPanel;
    private _resource?: vscode.Uri;


    constructor(private _context: vscode.ExtensionContext) {
        super(() => {
          this.dispose();
        });
    
        _context.subscriptions.push(
          vscode.window.onDidChangeActiveTextEditor(editor => {
            if (this._panel && editor && isCDSWorkflowFile(editor.document)) {
              this.load(editor.document.uri);
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
    
      public load(resource: vscode.Uri) {
        this._resource = resource;
    
        // Create panel webview
        if (!this._panel) {
          this._panel = vscode.window.createWebviewPanel(
            CDSPreview.viewType,
            "CDS Workflow Preview",
            vscode.ViewColumn.Two,
            {
              enableScripts: true,
              localResourceRoots: [
                vscode.Uri.joinPath(this._context.extensionUri, dirWeb),
              ]
            }
          );
    
          this._panel.onDidDispose(() => {
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
    
          const resourceLabel = uri.Utils.basename(this._resource);
          this._panel.title = 'Preview ' + resourceLabel;
          this._panel.webview.html = this.getHtmlContent();
        } else {
          this.refresh();
        }
      }

      // Refresh the webview
      public refresh() {
        if (this._panel && this._resource) {
          vscode.workspace.openTextDocument(this._resource).then(document => {
            this._panel?.webview.postMessage({
              type: 'refresh',
              value: document.getText(),
            });
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
    
        return `<!doctype html>
          <html lang="en">
          <head>
            <meta charset="utf-8">
            <title>CDS.Preview</title>
            <base href="${baseUri}/">
            <meta name="viewport" content="width=device-width, initial-scale=1">
            <link rel="stylesheet" href="${stylesUri}"></head>
          <body>
            <app-root></app-root>
            <script src="${scriptPolyfillsUri}" type="module"></script>
            <script src="${scriptMainUri}" type="module"></script>
            <script src="${scriptRuntimeUri}" type="module"></script>
            
          </body>
          </html>`;
      }
}