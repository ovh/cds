export enum BookmarkType {
    Project = "project",
    Workflow = "workflow",
    WorkflowLegacy = "workflow-legacy"
}

export class Bookmark {
    type: BookmarkType;
    id: string;
    label: string;
}