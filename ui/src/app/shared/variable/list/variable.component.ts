import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    Output,
    ViewChild
} from '@angular/core';
import { Project } from 'app/model/project.model';
import { Variable, VariableAudit } from 'app/model/variable.model';
import { ApplicationAuditService } from 'app/service/application/application.audit.service';
import { EnvironmentAuditService } from 'app/service/environment/environment.audit.service';
import { ProjectAuditService } from 'app/service/project/project.audit.service';
import { VariableService } from 'app/service/variable/variable.service';
import { Table } from 'app/shared/table/table';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import { SemanticModalComponent } from 'ng-semantic/ng-semantic';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-variable',
    templateUrl: './variable.html',
    styleUrls: ['./variable.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VariableComponent extends Table<Variable> {

    @Input()
    set variables(data: Variable[]) {
        this._variables = data;
        this.goTopage(1);
    }
    get variables() {
        return this._variables;
    }
    @Input()
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    }
    // display mode:   edit (edit all field) / launcher (only type value) /ro (display field, no edit)
    @Input() mode = 'edit';

    // project/application/environment
    @Input() auditContext: string;
    @Input() project: Project;
    @Input() environmentName: string;
    @Input() applicationName: string;

    @Output() event = new EventEmitter<VariableEvent>();

    @ViewChild('auditModal')
    auditModal: SemanticModalComponent;

    public ready = false;
    public variableTypes: string[];
    public currentVariableAudits: Array<VariableAudit>;
    private _variables: Variable[];
    filter: string;

    constructor(private _variableService: VariableService, private _projAudit: ProjectAuditService,
        private _envAudit: EnvironmentAuditService, private _appAudit: ApplicationAuditService, public _cd: ChangeDetectorRef) {
        super();
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

    getData(): Array<Variable> {
        if (!this.filter || this.filter === '') {
            return this.variables;
        } else {
            return this.variables.filter(v => v.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
        }
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
        if (this.auditModal) {
            this.currentVariableAudits = undefined;
            switch (this.auditContext) {
                case 'project':
                    this._projAudit.getVariableAudit(this.project.key, v.name).subscribe(audits => {
                        this.currentVariableAudits = audits;
                        setTimeout(() => {
                            this.auditModal.show({ observeChanges: true });
                        }, 100);
                    });
                    break;
                case 'environment':
                    this._envAudit.getVariableAudit(this.project.key, this.environmentName, v.name).subscribe(audits => {
                        this.currentVariableAudits = audits;
                        setTimeout(() => {
                            this.auditModal.show({ observeChanges: true });
                        }, 100);
                    });
                    break;
                case 'application':
                    this._appAudit.getVariableAudit(this.project.key, this.applicationName, v.name).subscribe(audits => {
                        this.currentVariableAudits = audits;
                        setTimeout(() => {
                            this.auditModal.show({ observeChanges: true });
                        }, 100);
                    });
                    break;
            }
        }

    }

}
