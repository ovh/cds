import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { VariableAudit } from 'app/model/variable.model';
import { Table } from 'app/shared/table/table';

@Component({
    selector: 'app-variable-audit',
    templateUrl: './variable.audit.html',
    styleUrls: ['./variable.audit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VariableAuditComponent extends Table<VariableAudit> {

    @Input() audits: Array<VariableAudit>;

    constructor() {
        super();
        this.nbElementsByPage = 8;
    }

    getData(): Array<VariableAudit> {
        return this.audits;
    }
}
