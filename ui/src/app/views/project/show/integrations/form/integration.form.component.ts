import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { finalize, first } from 'rxjs/operators';
import { IntegrationModel, ProjectIntegration } from '../../../../../model/integration.model';
import { Project } from '../../../../../model/project.model';
import { IntegrationService } from '../../../../../service/integration/integration.service';
import { ProjectStore } from '../../../../../service/project/project.store';
import { ToastService } from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-integration-form',
    templateUrl: './project.integration.form.html',
    styleUrls: ['./project.integration.form.scss']
})
export class ProjectIntegrationFormComponent {

    @Input() project: Project;
    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    models: Array<IntegrationModel>;
    newIntegration: ProjectIntegration;
    loading = false;
    codeMirrorConfig: {};

    constructor(private _integrationService: IntegrationService, private _projectStore: ProjectStore,
                private _toast: ToastService, private _translate: TranslateService) {
        this.newIntegration = new ProjectIntegration();
        this._integrationService.getIntegrationModels().pipe(first()).subscribe(platfs => {
            this.models = platfs.filter(pf => {
                return !pf.public;
            });
        });
        this.codeMirrorConfig = {
            mode: 'shell',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    updateConfig(): void {
        ProjectIntegration.mergeConfig(this.newIntegration.model.default_config, this.newIntegration.config);
    }

    create(): void {
        this.loading = true;
        this._projectStore.addIntegration(this.project.key, this.newIntegration)
          .pipe(
            first(),
            finalize(() => this.loading = false)
          ).subscribe(() => {
            this.newIntegration = new ProjectIntegration();
            this._toast.success('', this._translate.instant('project_updated'));
          });
    }
}
