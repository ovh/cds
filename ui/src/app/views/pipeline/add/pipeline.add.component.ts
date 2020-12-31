import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddPipeline, ImportPipeline } from 'app/store/pipelines.action';
import { finalize } from 'rxjs/operators';
import { Pipeline } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { ToastService } from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-pipeline-add',
    templateUrl: './pipeline.add.html',
    styleUrls: ['./pipeline.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class PipelineAddComponent {
    loadingCreate = false;
    newPipeline = new Pipeline();
    asCode = false;
    updated = false;

    codeMirrorConfig: any;
    pipToImport = `# Pipeline example
version: v1.0
name: root
jobs:
- job: run
  stage: Stage 1
  steps:
  - script:
    - echo "I'm the first step"
`;

    pipelineNamePattern = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    pipPatternError = false;

    project: Project;

    constructor(
        private store: Store,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._routeActivated.snapshot.data['project'];

        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };
    }

    createPipeline(): void {
        if (!this.newPipeline.name) {
            return;
        }

        if (!this.pipelineNamePattern.test(this.newPipeline.name)) {
            this.pipPatternError = true;
            return;
        }

        this.loadingCreate = true;
        this.store.dispatch(new AddPipeline({
            projectKey: this.project.key,
            pipeline: this.newPipeline
        })).pipe(finalize(() => {
            this.loadingCreate = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('pipeline_added'));
                this._router.navigate(['/project', this.project.key, 'pipeline', this.newPipeline.name]);
            });

    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'pipelines' } });
    }

    importPipeline() {
        this.loadingCreate = true;
        this.store.dispatch(new ImportPipeline({
            projectKey: this.project.key,
            pipelineCode: this.pipToImport
        })).pipe(finalize(() => {
            this.loadingCreate = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('pipeline_added'));
                this.goToProject();
            });
    }

    fileEvent(event: { content: string, file: File }) {
        this.pipToImport = event.content;
    }
}
