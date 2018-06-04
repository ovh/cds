import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {WarningModalComponent} from '../../../../shared/modal/warning/warning.component';
import {Project} from '../../../../model/project.model';
import {PermissionValue} from '../../../../model/permission.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {VariableEvent} from '../../../../shared/variable/variable.event.model';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {finalize, first} from 'rxjs/operators';
import {Warning} from '../../../../model/warning.model';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-project-variables',
    templateUrl: './variable.list.html',
    styleUrls: ['./variable.list.scss']
})
export class ProjectVariablesComponent implements OnInit {

    @Input() project: Project;
    @Input('warnings')
    set warnings(data: Array<Warning>) {
        if (data) {
            this.variableWarning = new Map<string, Warning>();
            this.unusedVariableWarning = new Array<Warning>();
            data.forEach(v => {
                let w = cloneDeep(v);
                w.element = w.element.replace('cds.proj.', '');
                if (w.type.indexOf('MISSING') !== -1) {
                    this.unusedVariableWarning.push(w);
                } else {
                    this.variableWarning.set(w.element, w);
                }
            });
        }
    };
    variableWarning: Map<string, Warning>;
    unusedVariableWarning: Array<Warning>;

    @ViewChild('varWarning')
    varWarningModal: WarningModalComponent;

    permissionEnum = PermissionValue;
    loading = true;
    varFormLoading = false;

    constructor(private _projectStore: ProjectStore,
                private _translate: TranslateService,
                private _toast: ToastService) {

    }

    ngOnInit() {
        if (this.project.variables) {
            this.loading = false;
            return;
        }
        this._projectStore.getProjectVariablesResolver(this.project.key)
            .pipe(first(), finalize(() => this.loading = false))
            .subscribe((proj) => {
                this.project = proj;
            });
    }

    variableEvent(event: VariableEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.varWarningModal.show(event);
        } else {
            event.variable.value = String(event.variable.value);
            switch (event.type) {
                case 'add':
                    this.varFormLoading = true;
                    this._projectStore.addProjectVariable(this.project.key, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_added'));
                        this.varFormLoading = false;
                    }, () => {
                        this.varFormLoading = false;
                    });
                    break;
                case 'update':
                    this._projectStore.updateProjectVariable(this.project.key, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_updated'));
                    });
                    break;
                case 'delete':
                    this._projectStore.deleteProjectVariable(this.project.key, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_deleted'));
                    });
                    break;
            }
        }
    }
}
