import {Component, Input, EventEmitter, Output, ViewChild} from '@angular/core';
import {Variable, VariableAudit} from '../../../model/variable.model';
import {SharedService} from '../../shared.service';
import {Table} from '../../table/table';
import {VariableService} from '../../../service/variable/variable.service';
import {VariableEvent} from '../variable.event.model';
import {ProjectAuditService} from '../../../service/project/project.audit.service';
import {Project} from '../../../model/project.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Environment} from '../../../model/environment.model';
import {Application} from '../../../model/application.model';
import {EnvironmentAuditService} from '../../../service/environment/environment.audit.service';
import {ApplicationAuditService} from '../../../service/application/application.audit.service';

@Component({
    selector: 'app-variable',
    templateUrl: './variable.html',
    styleUrls: ['./variable.scss']
})
export class VariableComponent extends Table {

    @Input() variables: Variable[];
    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };
    // display mode:   edit (edit all field) / launcher (only type value) /ro (display field, no edit)
    @Input() mode = 'edit';

    // project/application/environment
    @Input() auditContext: string;
    @Input() project: Project;
    @Input() environment: Environment;
    @Input() application: Application;

    @Output() event = new EventEmitter<VariableEvent>();

    @ViewChild('auditModal')
    auditModal: SemanticModalComponent;

    public ready = false;
    public variableTypes: string[];
    public currentVariableAudits: Array<VariableAudit>;
    filter: string;

    constructor(private _variableService: VariableService, private _sharedService: SharedService, private _projAudit: ProjectAuditService,
        private _envAudit: EnvironmentAuditService, private _appAudit: ApplicationAuditService) {
        super();
        this.variableTypes = this._variableService.getTypesFromCache();
        if (!this.variableTypes) {
            this._variableService.getTypesFromAPI().subscribe(types => {
                this.variableTypes = types;
                this.ready = true;
            });
        } else {
            this.ready = true;
        }
    }

    getData(): any[] {
        if (!this.filter || this.filter === '') {
            return this.variables;
        } else {
            return this.variables.filter(v => v.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
        }
    }

    /**
     * Send Event to parent component.
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
                            this.auditModal.show({observeChanges: true});
                        }, 100);
                    });
                    break;
                case 'environment':
                    this._envAudit.getVariableAudit(this.project.key, this.environment.name, v.name).subscribe(audits => {
                        this.currentVariableAudits = audits;
                        setTimeout(() => {
                            this.auditModal.show({observeChanges: true});
                        }, 100);
                    });
                    break;
                case 'application':
                    this._appAudit.getVariableAudit(this.project.key, this.application.name, v.name).subscribe(audits => {
                        this.currentVariableAudits = audits;
                        setTimeout(() => {
                            this.auditModal.show({observeChanges: true});
                        }, 100);
                    });
                    break;
            }
        }

    }

}
