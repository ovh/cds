import {Component, Input, Output, EventEmitter, OnInit} from '@angular/core';
import {Environment} from '../../../../../../model/environment.model';
import {User} from '../../../../../../model/user.model';
import {Workflow} from '../../../../../../model/workflow.model';
import {Project} from '../../../../../../model/project.model';
import {VariableEvent} from '../../../../../../shared/variable/variable.event.model';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {AuthentificationStore} from '../../../../../../service/auth/authentification.store';
import {EnvironmentService} from '../../../../../../service/environment/environment.service';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {cloneDeep} from 'lodash';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-environment',
    templateUrl: './environment.html',
    styleUrls: ['./environment.scss']
})
export class ProjectEnvironmentComponent {

    editableEnvironment: Environment;
    attachedWorkflows: Array<Workflow> = [];
    oldEnvName: string;
    cloneName: string;
    currentUser: User;

    hasChanged = false;
    loading = false;
    cloneLoading = false;
    addVarLoading = false;

    @Input('environment')
    set environment(data: Environment) {
        if (!data) {
            return;
        }
        this.oldEnvName = data.name;
        this.editableEnvironment = cloneDeep(data);
        if (data.usage) {
            this.attachedWorkflows = data.usage.workflows || [];
        }
    }
    get environment() {
        return this.editableEnvironment;
    }
    @Input() project: Project;

    @Output() deletedEnv = new EventEmitter<string>();

    constructor(private _projectStore: ProjectStore, private _toast: ToastService,
      private _translate: TranslateService, private _authenticationStore: AuthentificationStore) {
          this.currentUser = this._authenticationStore.getUser();
    }

    renameEnvironment(): void {
        this.loading = true;
        this._projectStore.renameProjectEnvironment(this.project.key, this.oldEnvName, this.editableEnvironment)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('environment_renamed'));
            });
    }

    cloneEnvironment(cloneModal?: any): void {
        this.cloneLoading = true;

        this._projectStore.cloneProjectEnvironment(this.project.key, this.editableEnvironment, this.cloneName)
            .pipe(finalize(() => {
                this.cloneLoading = false;
                this.cloneName = '';
                cloneModal.hide();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('environment_cloned')));
    }

    deleteEnvironment(): void {
        this.loading = true;
        this._projectStore.deleteProjectEnvironment(this.project.key, this.editableEnvironment)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('environment_deleted'));
                this.deletedEnv.emit(this.editableEnvironment.name);
            });
    }


    variableEvent(event: VariableEvent): void {
        event.variable.value = String(event.variable.value);
        switch (event.type) {
            case 'add':
                this.addVarLoading = true;
                this._projectStore.addEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .pipe(finalize(() => this.addVarLoading = false))
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_added'));
                    });
                break;
            case 'update':
                this._projectStore.updateEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .pipe(finalize(() => event.variable.updating = false))
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_updated'));
                    });
                break;
            case 'delete':
                this._projectStore.removeEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .pipe(finalize(() => event.variable.updating = false))
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_deleted'));
                    });
                break;
        }
    }
}
