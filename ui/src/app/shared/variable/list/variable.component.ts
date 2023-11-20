import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    Output
} from '@angular/core';
import { Project } from 'app/model/project.model';
import { Variable, VariableAudit } from 'app/model/variable.model';
import { ApplicationAuditService } from 'app/service/application/application.audit.service';
import { EnvironmentAuditService } from 'app/service/environment/environment.audit.service';
import { ProjectAuditService } from 'app/service/project/project.audit.service';
import { VariableService } from 'app/service/variable/variable.service';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import { finalize } from 'rxjs/operators';
import { NzModalService } from 'ng-zorro-antd/modal';
import { VariableAuditComponent } from 'app/shared/variable/audit/audit.component';

@Component({
    selector: 'app-variable',
    templateUrl: './variable.html',
    styleUrls: ['./variable.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VariableComponent {

    @Input()
    set variables(data: Variable[]) {
        this._variables = data;
        this.filterVariables();
    }
    get variables() {
        return this._variables;
    }

    // display mode:   edit (edit all field) / launcher (only type value) /ro (display field, no edit)
    @Input() mode = 'edit';

    // project/application/environment
    @Input() auditContext: string;
    @Input() project: Project;
    @Input() environmentName: string;
    @Input() applicationName: string;

    @Output() event = new EventEmitter<VariableEvent>();

    public ready = false;
    public variableTypes: string[];
    private _variables: Variable[];
    filteredVariables: Variable[] = [];
    filter: string;

    constructor(private _variableService: VariableService, private _projAudit: ProjectAuditService, private _modalService: NzModalService,
        private _envAudit: EnvironmentAuditService, private _appAudit: ApplicationAuditService, public _cd: ChangeDetectorRef) {
        this.variableTypes = this._variableService.getTypesFromCache();
        if (!this.variableTypes) {
            this._variableService.getTypesFromAPI().pipe(finalize(() => {
                this.ready = true;
                this._cd.detectChanges();
            })).subscribe(types => {
                this.variableTypes = types;
            });
        } else {
            this.ready = true;
        }
    }

    filterVariables(): void {
        if (!this.filter || this.filter === '') {
            this.filteredVariables = Object.assign([], this.variables);
        } else {
            this.filteredVariables = Object.assign([], this.variables.filter(v => v.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1));
        }
        this._cd.markForCheck();
    }

    /**
     * Send Event to parent component.
     *
     * @param type Type of event (update, delete)
     * @param variable Variable data
     */
    sendEvent(type: string, variable: Variable): void {
        variable.updating = true;
        this.event.emit(new VariableEvent(type, variable));
    }

    /**
     * Open audit modal
     */
    showAudit(event: any, v: Variable): void {
        switch (this.auditContext) {
            case 'project':
                this._projAudit.getVariableAudit(this.project.key, v.name).subscribe(audits => {
                    this.openAuditModal(audits);
                });
                break;
            case 'environment':
                this._envAudit.getVariableAudit(this.project.key, this.environmentName, v.name).subscribe(audits => {
                    this.openAuditModal(audits);
                });
                break;
            case 'application':
                this._appAudit.getVariableAudit(this.project.key, this.applicationName, v.name).subscribe(audits => {
                    this.openAuditModal(audits);
                })
                break;
            }
    }

    openAuditModal(audits: Array<VariableAudit>): void {
        this._modalService.create({
            nzWidth: '900px',
            nzTitle: 'Variable audit',
            nzContent: VariableAuditComponent,
            nzData: {
                audits: audits
            }
        });
    }

}
