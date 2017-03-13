import {Component, Input} from '@angular/core';
import {VariableAudit} from '../../../model/variable.model';
import {Table} from '../../table/table';

@Component({
    selector: 'app-variable_audit',
    templateUrl: './variable.audit.html',
    styleUrls: ['./variable.audit.scss']
})
export class VariableAuditComponent extends Table {

    @Input() audits: Array<VariableAudit>;

    constructor() {
        super();
        this.nbElementsByPage = 8;
    }

    getData(): any[] {
        return this.audits;
    }
}