import { Component, EventEmitter, Input, Output } from '@angular/core';
import { AuditWorkflow } from '../../../model/audit.model';
import { Item } from '../../diff/list/diff.list.component';
import { Table } from '../../table/table';

@Component({
    selector: 'app-audit-list',
    templateUrl: './audit.list.html',
    styleUrls: ['./audit.list.scss']
})
export class AuditListComponent extends Table {
    @Input() audits: Array<AuditWorkflow>;
    @Output() rollback: EventEmitter<number> = new EventEmitter();
    selectedAudit: AuditWorkflow;
    diffType: string;
    items: Array<Item>;

    getData(): any[] {
        return this.audits;
    }

    constructor() {
        super();
    }

    updateSelectedAudit(a: AuditWorkflow): void {
        this.selectedAudit = a;

        switch (a.data_type) {
            case 'yaml':
                this.diffType = 'text/x-yaml';
                break;
            case 'json':
                this.diffType = 'application/json';
                break;
            default:
                this.diffType = 'text/plain';
        }

        this.items = [<Item>{
            before: this.selectedAudit.data_before,
            after: this.selectedAudit.data_after,
            type: this.diffType,
        }]
    }
}
