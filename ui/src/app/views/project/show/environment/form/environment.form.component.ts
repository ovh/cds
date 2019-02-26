import { Component, EventEmitter, Input, Output } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddEnvironmentInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import { Environment } from '../../../../../model/environment.model';
import { Project } from '../../../../../model/project.model';
import { ToastService } from '../../../../../shared/toast/ToastService';

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

    constructor(private store: Store, private _toast: ToastService, private _translate: TranslateService) { }


    createEnv(): void {
        if (this.newEnvironment.name !== '') {
            this.loading = true;
            this.store.dispatch(new AddEnvironmentInProject({ projectKey: this.project.key, environment: this.newEnvironment }))
                .pipe(finalize(() => this.loading = false))
                .subscribe(() => {
                    this._toast.success('', this._translate.instant('environment_created'));
                    this.envCreated.emit(this.newEnvironment.name);
                    this.newEnvironment = new Environment();
                });
        }
    }
}
