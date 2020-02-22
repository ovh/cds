import * as vscode from 'vscode';

export type Dictionary<T> = {
    [key: string]: T;
};

export module Dictionary {
    export function of<T>(): Dictionary<T> {
        return {};
    }
}

export class ResourceKind implements vscode.QuickPickItem {
    constructor(readonly displayName: string, readonly pluralDisplayName: string, readonly manifestKind: string, readonly abbreviation: string) {
    }

    get label() { return this.displayName; }
    get description() { return ''; }
}

export const allKinds: Dictionary<ResourceKind> = {
    application: new ResourceKind("Application", "Applications", "Application", "application"),
    favoriteProject: new ResourceKind("⭐Project", "⭐Projects", "Project", "workflow"),
    favoriteWorkflow: new ResourceKind("⭐Workflow", "⭐Workflows", "Workflow", "workflow"),
    pipeline: new ResourceKind("Pipeline", "Pipelines", "Pipeline", "pipeline"),
    stage: new ResourceKind("Stage", "Stages", "Stage", "stage"),
    project: new ResourceKind("Project", "Projects", "Project", "project"),
    workflow: new ResourceKind("Workflow", "Workflows", "Workflow", "workflow"),
    workflowRun: new ResourceKind("Workflow Run", "Workflow Runs", "WorkflowRun", "workflowRun"),
    building: new ResourceKind("Building", "Building", "building", "building"),
    waiting: new ResourceKind("Waiting", "Waiting", "waiting", "waiting")
};
