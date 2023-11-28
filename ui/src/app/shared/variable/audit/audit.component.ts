import {ChangeDetectionStrategy, Component, inject, Input} from '@angular/core';
import { VariableAudit } from 'app/model/variable.model';
import { Table } from 'app/shared/table/table';
import {NZ_MODAL_DATA, NzModalRef} from "ng-zorro-antd/modal";

interface IModalData {
    audits: Array<VariableAudit>;
}

@Component({
    selector: 'app-variable-audit',
    templateUrl: './variable.audit.html',
    styleUrls: ['./variable.audit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VariableAuditComponent extends Table<VariableAudit> {

    readonly nzModalData: IModalData = inject(NZ_MODAL_DATA);

    constructor(private _modal: NzModalRef) {
        super();
        this.nbElementsByPage = 8;
    }

    getData(): Array<VariableAudit> {
        return this.nzModalData.audits;
    }
}
