import {Component, Input, Output, EventEmitter} from '@angular/core';
import {Environment} from '../../../../../model/environment.model';
import {ProjectStore} from '../../../../../service/project/project.store';
import {Project} from '../../../../../model/project.model';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';

@Component({
    selector: 'app-environment-form',
    templateUrl: './environment.form.html',
    styleUrls: ['./environment.form.scss']
})
export class ProjectEnvironmentFormComponent {

    @Input() project: Project;
    @Output() envCreated = new EventEmitter<string>();

    newEnvironment: Environment = new Environment();
    loading = false;

    constructor(private _projectStore: ProjectStore, private _toast: ToastService, private _translate: TranslateService) { }


    createEnv(): void {
        if (this.newEnvironment.name !== '') {
            this.loading = true;
            this._projectStore.addProjectEnvironment(this.project.key, this.newEnvironment).subscribe(() => {
                this._toast.success('', this._translate.instant('environment_created'));
                this.loading = false;
                this.project.environments.push(this.newEnvironment);
                this.envCreated.emit(this.newEnvironment.name);
                this.newEnvironment = new Environment();
            }, () => {
                this.loading = false;
            });
        }
    }
}
