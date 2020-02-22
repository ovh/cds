import * as vscode from "vscode";
import { CDSExt } from './cdsext';
import { CDSObject, CDSResourceFolder, DummyObject, ResourceNode } from './view.explorer';
import { WorkflowNodeJobRun } from "./models/workflow_run";

export async function refreshExplorerQueue(): Promise<void> {
    await vscode.commands.executeCommand('extension.vsCdsRefreshExplorerQueue');
}

export class CDSExplorerQueue implements vscode.TreeDataProvider<CDSObject> {
    private static instance: CDSExplorerQueue;
    private onDidChangeTreeDataEmitter: vscode.EventEmitter<CDSObject | undefined> = new vscode.EventEmitter<CDSObject | undefined>();
    readonly onDidChangeTreeData: vscode.Event<CDSObject | undefined> = this.onDidChangeTreeDataEmitter.event;

    public static getInstance(): CDSExplorerQueue {
        if (!this.instance) {
            this.instance = new CDSExplorerQueue();
        }
        return this.instance;
    }

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
        return jobs.map((job) => new CDSQueueJobNode(this.getLabel(job), job));
    }

    private getLabel(job: WorkflowNodeJobRun): string {
        return `${job.since} ${job.project_key}/${job.workflow_name}`;
    }
}

class CDSQueueJobNode implements CDSObject {
    constructor(readonly label: string, readonly metadata: WorkflowNodeJobRun) {}

    async getChildren(): Promise<CDSObject[]> {
        return [
            new CDSQueueJoDetailbNode(`Triggered By: ${this.metadata.triggered_by}`),
            new CDSQueueJoDetailbNode(`Project: ${this.metadata.project_key}`),
            new CDSQueueJoDetailbNode(`Workflow: ${this.metadata.workflow_name}`),
            new CDSQueueJoDetailbNode(`Pipeline: ${this.metadata.pipeline_name}`),
            new CDSQueueJoDetailbNode(`Run: ${this.metadata.run}`),
            new CDSQueueJoDetailbNode(`Booked by: ${this.metadata.booked_by}`)
        ];
    }

    public getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.label, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.workflowRun";
        return treeItem;
    }

    uri(): vscode.Uri {
        return vscode.Uri.parse(this.metadata.url);
    }
}

class CDSQueueJoDetailbNode implements CDSObject, ResourceNode {
    constructor(readonly label: string) {
    }

    async getChildren(): Promise<CDSObject[]> {
        return [new DummyObject("Not implemented")];
    }

    public uri(): vscode.Uri {
        return vscode.Uri.parse('Not implemented');
    }

    getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.label, vscode.TreeItemCollapsibleState.None);
        return treeItem;
    }
}