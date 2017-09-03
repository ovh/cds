import {Component, Input, Output, EventEmitter} from '@angular/core';
import {Environment} from '../../../../../../model/environment.model';
import {Project} from '../../../../../../model/project.model';
import {VariableEvent} from '../../../../../../shared/variable/variable.event.model';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-environment',
    templateUrl: './environment.html',
    styleUrls: ['./environment.scss']
})
export class ProjectEnvironmentComponent {

    editableEnvironment: Environment;
    oldEnvName: string;
    cloneName: string;

    hasChanged = false;
    loading = false;
    cloneLoading = false;
    addVarLoading = false;

    @Input('environment')
    set environment(data: Environment) {
        this.oldEnvName = data.name;
        this.editableEnvironment = cloneDeep(data);
    }
    @Input() project: Project;

    @Output() deletedEnv = new EventEmitter<string>();

    constructor(private _projectStore: ProjectStore, private _toast: ToastService, private _translate: TranslateService) {
    }

    renameEnvironment(): void {
        this.loading = true;
        this._projectStore.renameProjectEnvironment(this.project.key, this.oldEnvName, this.editableEnvironment)
            .finally(() => this.loading = false)
            .subscribe(() => {
                this._toast.success('', this._translate.instant('environment_renamed'));
            });
    }

    cloneEnvironment(cloneModal?: any): void {
        this.cloneLoading = true;

        this._projectStore.cloneProjectEnvironment(this.project.key, this.editableEnvironment, this.cloneName)
            .finally(() => {
                this.cloneLoading = false;
                this.cloneName = '';
                cloneModal.hide();
            })
            .subscribe(() => this._toast.success('', this._translate.instant('environment_cloned')));
    }

    deleteEnvironment(): void {
        this.loading = true;
        this._projectStore.deleteProjectEnvironment(this.project.key, this.editableEnvironment)
            .finally(() => this.loading = false)
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
                    .finally(() => this.addVarLoading = false)
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_added'));
                    });
                break;
            case 'update':
                this._projectStore.updateEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .finally(() => event.variable.updating = false)
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_updated'));
                    });
                break;
            case 'delete':
                this._projectStore.removeEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .finally(() => event.variable.updating = false)
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_deleted'));
                    });
                break;
        }
    }
}
