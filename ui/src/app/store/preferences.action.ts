export class SavePanelSize {
    static readonly type = '[Preferences] Save panel size';
    constructor(public payload: { panelKey: string, size: string }) { }
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
    constructor(public payload: { projectKey: string, name: string, value: string, sort: string }) { }
}

export class SaveProjectTreeExpandState {
    static readonly type = '[Preferences] Save project\'s explore tree expand state';
    constructor(public payload: { projectKey: string, state: { [key: string]: boolean } }) { }
}

export class SaveProjectBranchSelectState {
    static readonly type = '[Preferences] Save project\'s explore branch select state';
    constructor(public payload: { projectKey: string, state: { [key: string]: string } }) { }
}

export class DeleteProjectWorkflowRunFilter {
    static readonly type = '[Preferences] Delete project\'s workflow run filter';
    constructor(public payload: { projectKey: string, name: string }) { }
}

export class SaveMessageState {
    static readonly type = '[Preferences] Save message\'s state';
    constructor(public payload: { messageKey: string, value: boolean }) { }
}