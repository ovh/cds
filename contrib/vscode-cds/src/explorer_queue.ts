import * as vscode from "vscode";
import { CDSExt } from './cdsext';
import { CDSObject, CDSResourceFolder } from './explorer';
import { WorkflowNodeJobRun } from "./models/workflow_run";

export function createExplorerQueue(): CDSExplorerQueue {
    return new CDSExplorerQueue();
}

export async function refreshExplorerQueue(): Promise<void> {
    await vscode.commands.executeCommand('extension.vsCdsRefreshExplorerQueue');
}

export class CDSExplorerQueue implements vscode.TreeDataProvider<CDSObject> {
    private onDidChangeTreeDataEmitter: vscode.EventEmitter<CDSObject | undefined> = new vscode.EventEmitter<CDSObject | undefined>();
    readonly onDidChangeTreeData: vscode.Event<CDSObject | undefined> = this.onDidChangeTreeDataEmitter.event;

    constructor() {}

    public getTreeItem(element: CDSObject): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = element.getTreeItem();
        return treeItem;
    }

    public getChildren(parent?: CDSObject): vscode.ProviderResult<CDSObject[]> {
        return this.getChildrenBase(parent);
    }

    private getChildrenBase(parent?: CDSObject): vscode.ProviderResult<CDSObject[]> {
        if (parent) {
            return parent.getChildren();
        }
        return [
            new CDSQueueFolder("Waiting"),
            new CDSQueueFolder("Building"),
        ];
    }

    public refresh(): void {
        this.onDidChangeTreeDataEmitter.fire();
    }
}

class CDSQueueFolder extends CDSResourceFolder {
    constructor(readonly label: string) {
        super(label);
    }

    async getChildren(): Promise<CDSObject[]> {
        const jobs = await <Promise<WorkflowNodeJobRun[]>>CDSExt.getInstance().currentContext!.cdsctl.runCdsCommand(`queue --filter status=${this.label}`);
        return jobs.map((job) => new CDSQueueJobNode("TODO JOB Label", job));
    }
}

class CDSQueueJobNode implements CDSObject {
    constructor(readonly label: string, readonly metadata: WorkflowNodeJobRun) {}

    async getChildren(): Promise<CDSObject[]> {
        return [];
    }

    public getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem("TODO", vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.queue.Job";
        return treeItem;
    }

    uri(): vscode.Uri {
        // TODO
        return vscode.Uri.parse(CDSExt.getInstance().currentContext!.cdsctl.getConfigUiURL() + "/TODO/TODO");
    }
}
