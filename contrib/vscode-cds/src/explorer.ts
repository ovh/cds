import * as fs from 'fs';
import * as path from "path";
import * as toml from 'toml';
import * as vscode from "vscode";
import { CdsCtl } from "./cdsctl";
import { Application } from "./models/application";
import { Pipeline } from "./models/pipeline";
import { Project } from "./models/project";
import { WNode, Workflow } from "./models/workflow";
import { Action, Stage, WorkflowNodeJobRun, WorkflowNodeRun, WorkflowRun } from "./models/workflow_run";
import { allKinds, ResourceKind } from "./resources";
import { Property } from "./util.property";
import { CDSExt } from './cdsext';

export function createExplorer(): CDSExplorer {
    return new CDSExplorer();
}

export interface CDSObject {
    readonly nodeCategory: "cds-explorer-node";
    readonly nodeType: CDSExplorerNodeType;
    readonly id: string;
    readonly metadata?: any;
    getChildren(): vscode.ProviderResult<CDSObject[]>;
    getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem>;
}

export interface CDSContext {
    readonly name: string;
    readonly cdsctl: CdsCtl;
}

export async function refreshExplorer(): Promise<void> {
    await vscode.commands.executeCommand('extension.vsCdsRefreshExplorer');
}

export class CDSExplorer implements vscode.TreeDataProvider<CDSObject> {
    private onDidChangeTreeDataEmitter: vscode.EventEmitter<CDSObject | undefined> = new vscode.EventEmitter<CDSObject | undefined>();
    readonly onDidChangeTreeData: vscode.Event<CDSObject | undefined> = this.onDidChangeTreeDataEmitter.event;
    private contexts: Promise<CDSContext[]>;

    constructor() {
        this.contexts = discoverContexts();
    }

    public getTreeItem(element: CDSObject): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = element.getTreeItem();
        return treeItem;
    }

    public getChildren(parent?: CDSObject): vscode.ProviderResult<CDSObject[]> {
        return this.getChildrenBase(parent);
    }

    public refresh(): void {
        this.contexts = discoverContexts();
        this.onDidChangeTreeDataEmitter.fire();
    }

    private getChildrenBase(parent?: CDSObject): vscode.ProviderResult<CDSObject[]> {
        if (parent) {
            return parent.getChildren();
        }
        return this.getContextsNode();
    }

    private async getContextsNode(): Promise<CDSObject[]> {
        return (await this.contexts).map((context) => {
            return new CDSContextNode(context.cdsctl.getContextName(), context);
        });
    }
}

export const CDS_EXPLORER_NODE_CATEGORY = 'cds-explorer-node';
export type CDSExplorerNodeType =
    'error' |
    'resource' |
    'folder.resource' |
    'context';

const statusIcon: { [name: string]: string } = {
    Pending: "↻",
    Waiting: "↻",
    Building: "↻",
    Success: "✓",
    Fail: "✗",
    Disabled: "✓",
    "Never Built": "✓",
    Unknown: "✓",
    Skipped: "✓",
    Stopped: "✗",
    Stopping: "✗"
};

class CDSExplorerNodeImpl {
    readonly nodeCategory = CDS_EXPLORER_NODE_CATEGORY;
    constructor(readonly nodeType: CDSExplorerNodeType) { }
}

class CDSContextNode extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly id: string, readonly metadata: CDSContext) {
        super('context');
    }

    get icon(): vscode.Uri | undefined {
        if (CDSExt.getInstance().currentContext && CDSExt.getInstance().currentContext!.name === this.id) {
            return vscode.Uri.file(path.join(__dirname, "../images/cds.svg"));
        }
    }

    public getChildren(): vscode.ProviderResult<CDSObject[]> {
        return [
            new CDSWorkflowsFolder(allKinds.favoriteWorkflow, this.metadata),
            new CDSProjectsFolder(allKinds.favoriteProject, this.metadata),
        ];
    }

    public getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.iconPath = this.icon;
        treeItem.contextValue = "vsCds.context";
        return treeItem;
    }
}

export async function discoverContexts(): Promise<CDSContext[]> {
    const cdsConfigs = Property.get("knownCdsconfigs");
    if (!cdsConfigs) {
        return [];
    }

    const allContextes: CDSContext[][] = await Promise.all(cdsConfigs.map(async (configFile): Promise<CDSContext[]> => {
        const config = toml.parse(fs.readFileSync(configFile, 'utf-8'));
        const ctxs = new Array<CDSContext>();
        let current: string = "";

        for (const contextName in config) {
            if (contextName !== "current") {
                const cdsctl = new CdsCtl(configFile, contextName);
                ctxs.push({
                    name: contextName,
                    cdsctl,
                });
                if (current === contextName && !CDSExt.getInstance().currentContext) {
                    CDSExt.getInstance().currentContext = {name: contextName, cdsctl};
                }
            } else {
                current = config["current"];
            }
        }
        return ctxs;
    }));

    const results = new Array<CDSContext>();
    allContextes.forEach((ctxs) => {
        ctxs.forEach((ctx) => {
            results.push(ctx);
        });
    });

    return results;
}

export interface ResourceFolder {
    readonly kind: ResourceKind;
}

export interface ResourceNode {
    readonly kind: ResourceKind;
    readonly id: string;
    uri(): vscode.Uri;
}

abstract class CDSFolder extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly nodeType: CDSExplorerNodeType, readonly id: string, readonly displayName: string, readonly contextValue?: string) {
        super(nodeType);
    }

    abstract getChildren(): vscode.ProviderResult<CDSObject[]>;

    getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.displayName, vscode.TreeItemCollapsibleState.Collapsed);
        return treeItem;
    }
}

class CDSResourceFolder extends CDSFolder implements ResourceFolder {
    constructor(cdsContext: CDSContext, readonly kind: ResourceKind) {
        super("folder.resource", kind.abbreviation, kind.pluralDisplayName, "vscds.kind");
    }

    async getChildren(): Promise<CDSObject[]> {
        return [new DummyObject("Not implemented")];
    }
}

export class CDSResource extends CDSExplorerNodeImpl implements CDSObject, ResourceNode {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly metadata?: any) {
        super('resource');
    }

    async getChildren(): Promise<CDSObject[]> {
        return [new DummyObject("Not implemented")];
    }

    public uri(): vscode.Uri {
        return vscode.Uri.parse('Not implemented');
    }

    getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.None);
        return treeItem;
    }
}

class CDSWorkflowsFolder extends CDSResourceFolder {
    constructor(readonly kind: ResourceKind, readonly cdsContext: CDSContext, readonly projectKey?: string) {
        super(cdsContext, kind);
    }

    async getChildren(): Promise<CDSObject[]> {
        if (this.kind === allKinds.favoriteWorkflow) {
            const wf = await <Promise<Workflow[]>>this.cdsContext.cdsctl.runCdsCommand("workflow favorites list");
            return wf.map((wf) => new CDSWorkflowResource(this.cdsContext, this.kind, wf.name, wf));
        }

        const wf = await <Promise<Workflow[]>>this.cdsContext.cdsctl.runCdsCommand("workflow list " + this.projectKey!);
        return wf.map((wf) => new CDSWorkflowResource(this.cdsContext, this.kind, wf.name, wf));

    }
}

class CDSWorkflowResource extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly workflow: Workflow) {
        super('folder.resource');
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.workflowEdit";
        if (this.workflow.name) {
            treeItem.tooltip = this.workflow.name;
        }
        return treeItem;
    }

    async getChildren(): Promise<CDSObject[]> {
        const projectKey = this.workflow.project_key;
        const workflowName = this.workflow.name;
        const wrs = await <Promise<WorkflowRun[]>>this.cdsContext.cdsctl.runCdsCommand(`workflow history ${projectKey} ${workflowName}`);
        return wrs.map((wr) => new CDSWorklowRunResource(this.kind, this.cdsContext,
            `run ${wr.num} - ${wr.status}`,
             wr, this.workflow));
    }

    uri(): vscode.Uri {
        const projectKey = this.workflow.project_key;
        const workflowName = this.workflow.name;
        return vscode.Uri.parse(this.cdsContext.cdsctl.getConfigUiURL()
            + `/project/${projectKey}/workflow/${workflowName}`);
    }
}

class CDSWorklowRunResource extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly kind: ResourceKind, readonly cdsContext: CDSContext, readonly id: string, readonly workflowRun: WorkflowRun, readonly workflow: Workflow) {
        super('folder.resource');
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const icon = statusIcon[this.workflowRun.status];
        const line = `${icon} run ${this.workflowRun.num}`;
        const treeItem = new vscode.TreeItem(line, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.workflowRun";
        treeItem.tooltip = `${this.workflowRun.status} ${icon} run ${this.workflowRun.num}
start: ${this.workflowRun.start}
lastExecution: ${this.workflowRun.last_execution}`;
        if (this.workflowRun.tags) {
            this.workflowRun.tags.forEach((tag) => {
                treeItem.tooltip += `\n${tag.tag}: ${tag.value}`;
            });
        }
        return treeItem;
    }

    async getChildren(): Promise<CDSObject[]> {
        const rawCmd = this.cdsContext.cdsctl.buildRawCDSCommand(`admin curl /project/${this.workflow.project_key}/workflows/${this.workflow.name}/runs/${this.workflowRun.num}`);
        const w = await <Promise<any>>this.cdsContext.cdsctl.runCommand(rawCmd);
        const wr = JSON.parse(w);
        const workflowNodeRun = wr.nodes[wr.workflow.workflow_data.node.id][0];
        const title = `${statusIcon[workflowNodeRun.status]} ${wr.workflow.workflow_data.node.name}`;
        return [
            new CDSWorklowRunNodeResource(this.cdsContext, this.kind, title, wr, wr.workflow.workflow_data.node)
        ];
    }

    uri(): vscode.Uri {
        const projectKey = this.workflow.project_key;
        const workflowName = this.workflow.name;
        const num = this.workflowRun.num;
        return vscode.Uri.parse(this.cdsContext.cdsctl.getConfigUiURL()
            + `/project/${projectKey}/workflow/${workflowName}/run/${num}`);
    }
}

class CDSWorklowRunNodeResource extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly workflowRun: WorkflowRun, readonly workflowNode: WNode) {
        super('folder.resource');
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.workflowNodeRun";
        return treeItem;
    }

    getChildren(): vscode.ProviderResult<CDSObject[]> {
        const result = Array<CDSObject>();

        if (this.workflowNode.id in this.workflowRun.nodes) {
            const workflowNodeRun = this.workflowRun.nodes[this.workflowNode.id][0];
            result.push(new CDSWorklowPipelineStagesResource(this.cdsContext, allKinds.stage, "Stages", this.workflowRun, workflowNodeRun));
        }

        const nodesIDs = Array<number>();
        if (this.workflowNode.triggers) {
            for (const node of this.workflowNode.triggers) {
                nodesIDs.push(node.child_node_id);
                let title = `${node.child_node.name}`;
                if (node.child_node.id in this.workflowRun.nodes) {
                    const childNode = this.workflowRun.nodes[node.child_node.id][0];
                    title = `${statusIcon[childNode.status]} ${title}`;
                }
                result.push(new CDSWorklowRunNodeResource(this.cdsContext, this.kind, title, this.workflowRun, node.child_node));
            }
        }

        if (this.workflowRun.workflow.workflow_data.joins) {
            const nodesIDsAlreadyAddded = Array<number>(); // to avoid having multiple time the same join
            for (const node of this.workflowRun.workflow.workflow_data.joins) {
                for (const parent of node.parents) {
                    if (this.workflowNode.id === parent.parent_id && !nodesIDsAlreadyAddded.includes(node.id)) {
                        nodesIDsAlreadyAddded.push(node.id);
                        result.push(new CDSWorklowRunNodeResource(this.cdsContext, this.kind, node.name, this.workflowRun, node));
                    }
                }
            }
        }
        return result;
    }
}

class CDSWorklowPipelineStagesResource extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly workflowRun: WorkflowRun, readonly workflowNodeRun: WorkflowNodeRun) {
        super('folder.resource');
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.workflowStage";
        return treeItem;
    }

    getChildren(): vscode.ProviderResult<CDSObject[]> {
        const result = Array<CDSObject>();

        if (this.workflowNodeRun.stages) {
            for (const stage of this.workflowNodeRun.stages) {
                let stageName = stage.name;
                if (stageName === "") {
                    stageName = "stage " + stage.build_order;
                }
                const title = `${statusIcon[stage.status]} ${stageName}`;
                result.push(new CDSWorklowPipelineStageResource(this.cdsContext, allKinds.stage, title, this.workflowRun, stage));
            }
        }

        return result;
    }
}

class CDSWorklowPipelineStageResource extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly workflowRun: WorkflowRun, readonly stage: Stage) {
        super('folder.resource');
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.workflowStage";
        return treeItem;
    }

    getChildren(): vscode.ProviderResult<CDSObject[]> {
        const result = Array<CDSObject>();

        if (this.stage.run_jobs) {
            for (const job of this.stage.run_jobs) {
                const title = `${statusIcon[job.status]} ${job.job.action.name}`;
                result.push(new CDSWorklowPipelineJobResource(this.cdsContext, allKinds.stage, title, this.workflowRun, job));
            }
        }

        return result;
    }
}

class CDSWorklowPipelineJobResource extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly workflowRun: WorkflowRun, readonly job: WorkflowNodeJobRun) {
        super('folder.resource');
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.workflowJob";
        return treeItem;
    }

    getChildren(): vscode.ProviderResult<CDSObject[]> {
        const result = Array<CDSObject>();

        if (this.job.job.action.actions) {
            for (let index = 0; index < this.job.job.action.actions.length; index++) {
                const action = this.job.job.action.actions[index];
                let stepName = action.step_name||"";
                if (stepName === "") {
                    stepName = action.name;
                }

                let title = `${stepName}`;
                if (this.job.job.step_status) {
                    title = `${statusIcon[this.job.job.step_status[index].status]} ${title}`;
                }

                result.push(new CDSStepStatusResource(this.cdsContext, allKinds.stage, title, action));
            }
        }

        return result;
    }
}

class CDSStepStatusResource extends CDSResource {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly action: Action) {
        super(cdsContext, kind, id, action);
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = await super.getTreeItem();
        if (this.metadata.name) {
            treeItem.tooltip = this.id;
        }
        treeItem.contextValue = "vsCds.workflowStep";
        return treeItem;
    }
}

class CDSProjectsFolder extends CDSResourceFolder {
    constructor(readonly kind: ResourceKind, readonly cdsContext: CDSContext) {
        super(cdsContext, kind);
    }

    async getChildren(): Promise<CDSObject[]> {
        const prj = await <Promise<Project[]>>this.cdsContext.cdsctl.runCdsCommand("project favorites list");
        return prj.map((prj) => new CDSProjectResource(this.kind, this.cdsContext, prj.name, prj));
    }
}

class CDSProjectResource extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly kind: ResourceKind, readonly cdsContext: CDSContext, readonly id: string, readonly metadata?: any) {
        super('folder.resource');
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.Collapsed);
        treeItem.contextValue = "vsCds.project";
        return treeItem;
    }

    getChildren(): vscode.ProviderResult<CDSObject[]> {
        return [
            new CDSWorkflowsFolder(allKinds.workflow, this.cdsContext, this.metadata.key),
            new CDSApplicationsFolder(allKinds.application, this.cdsContext, this.metadata.key),
            new CDSPipelinesFolder(allKinds.pipeline, this.cdsContext, this.metadata.key),
        ];
    }

    uri(): vscode.Uri {
        const projectKey = this.metadata.project_key;
        return vscode.Uri.parse(this.cdsContext.cdsctl.getConfigUiURL()
            + `/project/${projectKey}`);
    }
}

class CDSApplicationsFolder extends CDSResourceFolder {
    constructor(readonly kind: ResourceKind, readonly cdsContext: CDSContext, readonly projectKey?: string) {
        super(cdsContext, kind);
    }

    async getChildren(): Promise<CDSObject[]> {
        const wf = await <Promise<Application[]>>this.cdsContext.cdsctl.runCdsCommand("application list " + this.projectKey!);
        return wf.map((wf) => new CDSApplicationResource(this.cdsContext, this.kind, wf.name, wf));
    }
}

class CDSApplicationResource extends CDSResource {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly metadata?: any) {
        super(cdsContext, kind, id, metadata);
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = await super.getTreeItem();
        if (this.metadata.name) {
            treeItem.tooltip = this.metadata.name;
        }
        treeItem.contextValue = "vsCds.application";
        return treeItem;
    }

    uri(): vscode.Uri {
        const projectKey = this.metadata.project_key;
        const applicationName = this.metadata.name;
        return vscode.Uri.parse(this.cdsContext.cdsctl.getConfigUiURL()
            + `/project/${projectKey}/application/${applicationName}`);
    }
}

class CDSPipelinesFolder extends CDSResourceFolder {
    constructor(readonly kind: ResourceKind, readonly cdsContext: CDSContext, readonly projectKey?: string) {
        super(cdsContext, kind);
    }

    async getChildren(): Promise<CDSObject[]> {
        const pipelines = await <Promise<Pipeline[]>>this.cdsContext.cdsctl.runCdsCommand("pipeline list " + this.projectKey!);
        return pipelines.map((pipeline) => new CDSPipelineResource(this.cdsContext, this.kind, pipeline.name, pipeline));
    }
}

class CDSPipelineResource extends CDSResource {
    constructor(readonly cdsContext: CDSContext, readonly kind: ResourceKind, readonly id: string, readonly metadata?: any) {
        super(cdsContext, kind, id, metadata);
    }

    async getTreeItem(): Promise<vscode.TreeItem> {
        const treeItem = await super.getTreeItem();
        if (this.metadata.name) {
            treeItem.tooltip = this.metadata.name;
        }
        treeItem.contextValue = "vsCds.pipeline";
        return treeItem;
    }

    uri(): vscode.Uri {
        const projectKey = this.metadata.project_key;
        const pipelineName = this.metadata.name;
        return vscode.Uri.parse(this.cdsContext.cdsctl.getConfigUiURL()
            + `/project/${projectKey}/application/${pipelineName}`);
    }
}

class DummyObject extends CDSExplorerNodeImpl implements CDSObject {
    constructor(readonly id: string, readonly diagnostic?: string) {
        super('error');
    }

    getTreeItem(): vscode.TreeItem | Thenable<vscode.TreeItem> {
        return new vscode.TreeItem(this.id, vscode.TreeItemCollapsibleState.None);
    }

    getChildren(): vscode.ProviderResult<CDSObject[]> {
        return [];
    }
}
