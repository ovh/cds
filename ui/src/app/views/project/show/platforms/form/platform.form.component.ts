import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { finalize, first } from 'rxjs/operators';
import { PlatformModel, ProjectPlatform } from '../../../../../model/platform.model';
import { Project } from '../../../../../model/project.model';
import { PlatformService } from '../../../../../service/platform/platform.service';
import { ProjectStore } from '../../../../../service/project/project.store';
import { ToastService } from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-platform-form',
    templateUrl: './project.platform.form.html',
    styleUrls: ['./project.platform.form.scss']
})
export class ProjectPlatformFormComponent {

    @Input() project: Project;
    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    models: Array<PlatformModel>;
    newPlatform: ProjectPlatform;
    loading = false;
    codeMirrorConfig: {};

    constructor(private _platformService: PlatformService, private _projectStore: ProjectStore,
                private _toast: ToastService, private _translate: TranslateService) {
        this.newPlatform = new ProjectPlatform();
        this._platformService.getPlatformModels().pipe(first()).subscribe(platfs => {
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
        ProjectPlatform.mergeConfig(this.newPlatform.model.default_config, this.newPlatform.config);
    }

    create(): void {
        this.loading = true;
        this._projectStore.addPlatform(this.project.key, this.newPlatform)
          .pipe(
            first(),
            finalize(() => this.loading = false)
          ).subscribe(() => {
            this.newPlatform = new ProjectPlatform();
            this._toast.success('', this._translate.instant('project_updated'));
          });
    }
}
