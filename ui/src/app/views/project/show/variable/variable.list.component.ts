import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddVariableInProject, DeleteVariableInProject, FetchVariablesInProject, UpdateVariableInProject } from 'app/store/project.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize } from 'rxjs/operators';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { Warning } from '../../../../model/warning.model';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../shared/toast/ToastService';
import { VariableEvent } from '../../../../shared/variable/variable.event.model';

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

    constructor(
        private _translate: TranslateService,
        private _toast: ToastService,
        private store: Store
    ) {

    }

    ngOnInit() {
        this.store.dispatch(new FetchVariablesInProject({ projectKey: this.project.key }))
            .pipe(finalize(() => this.loading = false))
            .subscribe();
    }

    variableEvent(event: VariableEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.varWarningModal.show(event);
        } else {
            event.variable.value = String(event.variable.value);
            switch (event.type) {
                case 'add':
                    this.varFormLoading = true;
                    this.store.dispatch(new AddVariableInProject(event.variable))
                        .pipe(finalize(() => this.varFormLoading = false))
                        .subscribe(() => this._toast.success('', this._translate.instant('variable_added')));
                    break;
                case 'update':
                    this.store.dispatch(new UpdateVariableInProject({ variableName: event.variable.name, changes: event.variable }))
                        .subscribe(() => this._toast.success('', this._translate.instant('variable_updated')));
                    break;
                case 'delete':
                    this.store.dispatch(new DeleteVariableInProject(event.variable))
                        .subscribe(() => this._toast.success('', this._translate.instant('variable_deleted')));
                    break;
            }
        }
    }
}
