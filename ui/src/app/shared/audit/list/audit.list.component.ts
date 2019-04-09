import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Column, ColumnType } from 'app/shared/table/data-table.component';
import { AuditWorkflow } from '../../../model/audit.model';
import { Item } from '../../diff/list/diff.list.component';

@Component({
    selector: 'app-audit-list',
    templateUrl: './audit.list.html',
    styleUrls: ['./audit.list.scss']
})
export class AuditListComponent {
    @Input() audits: Array<AuditWorkflow>;
    @Input() loading = false;
    @Output() rollback: EventEmitter<number> = new EventEmitter();
    selectedAudit: AuditWorkflow;
    diffType: string;
    items: Array<Item>;
    columns: Column<AuditWorkflow>[];

    constructor() {
        this.columns = [
            <Column<AuditWorkflow>>{
                type: ColumnType.TEXT,
                name: 'audit_action',
                selector: (audit: AuditWorkflow) => audit.event_type,
            },
            <Column<AuditWorkflow>>{
                type: ColumnType.TEXT,
                name: 'audit_username',
                selector: (audit: AuditWorkflow) => audit.triggered_by,
            },
            <Column<AuditWorkflow>>{
                type: ColumnType.DATE,
                name: 'audit_time_author',
                selector: (audit: AuditWorkflow) => audit.created,
            },
            <Column<AuditWorkflow>>{
                type: ColumnType.CONFIRM_BUTTON,
                name: '',
                selector: (audit: AuditWorkflow) => {
                    return {
                        title: 'common_rollback',
                        click: () => this.rollback.emit(audit.id)
                    };
                },
            },
        ];
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
