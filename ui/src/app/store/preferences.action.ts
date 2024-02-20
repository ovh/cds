export class SavePanelSize {
    static readonly type = '[Preferences] Save panel size';
    constructor(public payload: { panelKey: string, size: number }) { }
}

export class SetPanelResize {
    static readonly type = '[Preferences] Set panel resize';
    constructor(public payload: { resizing: boolean }) { }
}

export class SetTheme {
    static readonly type = '[Preferences] Set theme';
    constructor(public payload: { theme: string }) { }
}

export class SaveProjectWorkflowRunFilter {
    static readonly type = '[Preferences] Save project\'s workflow run filter';
    constructor(public payload: { projectKey: string, name: string, value: string }) { }
}

export class DeleteProjectWorkflowRunFilter {
    static readonly type = '[Preferences] Delete project\'s workflow run filter';
    constructor(public payload: { projectKey: string, name: string }) { }
}