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

export class SaveWorkflowRunSearch {
    static readonly type = '[Preferences] Save workflow run search';
    constructor(public payload: { name: string, value: string }) { }
}

export class DeleteWorkflowRunSearch {
    static readonly type = '[Preferences] Delete workflow run search';
    constructor(public payload: { name: string }) { }
}