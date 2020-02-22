import * as vscode from "vscode";
import { CDSExt } from './cdsext';
import { CDSObject } from './explorer';
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

    constructor() {
    }

    public getTreeItem(element: CDSObject): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = element.getTreeItem();
        return treeItem;
    }

    public getChildren(parent?: CDSObject): vscode.ProviderResult<CDSObject[]> {
        return this.getChildrenBase(parent);
    }

    public refresh(): void {
        this.onDidChangeTreeDataEmitter.fire();
    }

    private getChildrenBase(parent?: CDSObject): vscode.ProviderResult<CDSObject[]> {
        if (parent) {
            return parent.getChildren();
        }
        return this.getQueueJob();
    }

    private async getQueueJob(): Promise<CDSObject[]> {
        const jobs = await <Promise<WorkflowNodeJobRun[]>>CDSExt.getInstance().currentContext!.cdsctl.runCdsCommand("queue");
        return jobs.map((job) => new CDSQueueJobNode("TODO", job));
    }
}

class CDSExplorerQueueNodeImpl {
    constructor() { }
}

class CDSQueueJobNode extends CDSExplorerQueueNodeImpl implements CDSObject {
    constructor(readonly label: string, readonly metadata: WorkflowNodeJobRun) {
        super();
    }

    public getChildren(): vscode.ProviderResult<CDSObject[]> {
        return [];
    }

    public getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.metadata.status, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.queue.Job";
        return treeItem;
    }
}
