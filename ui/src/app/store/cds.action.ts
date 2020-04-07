
// Use to update maintenance state
export class UpdateMaintenance {
    static readonly type = '[CDS] Update Maintenance';
    constructor(public enable: boolean) { }
}

export class GetCDSStatus {
    static readonly type = '[CDS] Get CDS Status';
    constructor() { }
}


