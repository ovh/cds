import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import {
    AddVariableInProject,
    DeleteVariableInProject,
    FetchVariablesInProject,
    UpdateVariableInProject
} from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-variables',
    templateUrl: './variable.list.html',
    styleUrls: ['./variable.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectVariablesComponent implements OnInit {

    @Input() project: Project;
    @ViewChild('varWarning')
    varWarningModal: WarningModalComponent;

    loading = true;
    varFormLoading = false;

    constructor(
        private _translate: TranslateService,
        private _toast: ToastService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {

    }

    ngOnInit() {
        this.store.dispatch(new FetchVariablesInProject({ projectKey: this.project.key }))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
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
                        .pipe(finalize(() => {
                            this.varFormLoading = false;
                            this._cd.markForCheck();
                        }))
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
