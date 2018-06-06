import {Component, Input} from '@angular/core';
import {Table} from '../../table/table';
import {AuditWorkflow} from '../../../model/audit.model';

@Component({
    selector: 'app-audit-list',
    templateUrl: './audit.list.html',
    styleUrls: ['./audit.list.scss']
})
export class AuditListComponent extends Table {

    @Input() audits: Array<AuditWorkflow>;

    codeMirrorConfig: any;
    selectedAudit: AuditWorkflow;

    getData(): any[] {
        return this.audits;
    }

    constructor() {
        super();
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'text/x-yaml',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: 'nocursor'
        };
    }

    updateSelectedAudit(a: AuditWorkflow): void {
        console.log(a);
        this.selectedAudit = a;
        switch (a.data_type) {
            case 'yaml':
                this.codeMirrorConfig.mode = 'text/x-yaml';
                break;
            case 'json':
                this.codeMirrorConfig.mode = 'application/json';
                break;
        }
    }

}
