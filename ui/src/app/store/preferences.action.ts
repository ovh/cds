export class SavePanelSize {
    static readonly type = '[Preferences] Save panel size';
    constructor(public payload: { panelKey: string, size: number }) { }
}

export class SetTheme {
    static readonly type = '[Preferences] Set theme';
    constructor(public payload: { theme: string }) { }
}
