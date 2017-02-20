import {Component, Input} from '@angular/core';
import {Environment} from '../../../../../../model/environment.model';
import {Project} from '../../../../../../model/project.model';
import {VariableEvent} from '../../../../../../shared/variable/variable.event.model';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';

declare var _: any;

@Component({
    selector: 'app-environment',
    templateUrl: './environment.html',
    styleUrls: ['./environment.scss']
})
export class ProjectEnvironmentComponent {

    editableEnvironment: Environment;
    oldEnvName: string;

    hasChanged = false;
    loading = false;
    addVarLoading = false;

    @Input('environment')
    set environment(data: Environment) {
        this.oldEnvName = data.name;
        this.editableEnvironment = _.cloneDeep(data);
    }
    @Input() project: Project;

    constructor(private _projectStore: ProjectStore, private _toast: ToastService, private _translate: TranslateService) {
    }

    renameEnvironment(): void {
        this.loading = true;
        this._projectStore.renameProjectEnvironment(this.project.key, this.oldEnvName, this.editableEnvironment).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('environment_renamed'));
        }, () => {
            this.loading = false;
        });
    }

    deleteEnvironment(): void {
        this.loading = true;
        this._projectStore.deleteProjectEnvironment(this.project.key, this.editableEnvironment).subscribe(() => {
            this._toast.success('', this._translate.instant('environment_deleted'));
            this.loading = false;
        }, () => {
            this.loading = false;
        });
    }


    variableEvent(event: VariableEvent): void {
        event.variable.value = String(event.variable.value);
        switch (event.type) {
            case 'add':
                this.addVarLoading = true;
                this._projectStore.addEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .subscribe( () => {
                        this._toast.success('', this._translate.instant('variable_added'));
                            this.addVarLoading = false;
                    }, () => {
                        this.addVarLoading = false;
                    });
                break;
            case 'update':
                this._projectStore.updateEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .subscribe( () => {
                        this._toast.success('', this._translate.instant('variable_updated'));
                    });
                break;
            case 'delete':
                this._projectStore.removeEnvironmentVariable(this.project.key, this.editableEnvironment.name, event.variable)
                    .subscribe( () => {
                        this._toast.success('', this._translate.instant('variable_deleted'));
                    });
                break;
        }
    }
}
