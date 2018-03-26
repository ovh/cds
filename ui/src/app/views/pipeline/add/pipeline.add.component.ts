import {Component} from '@angular/core';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {Pipeline} from '../../../model/pipeline.model';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-pipeline-add',
    templateUrl: './pipeline.add.html',
    styleUrls: ['./pipeline.add.scss']
})
export class PipelineAddComponent {

    ready = false;
    loadingCreate = false;
    pipelineType: Array<string>;
    newPipeline = new Pipeline();

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

    pipelineNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    pipPatternError = false;

    project: Project;

    constructor(private _pipStore: PipelineStore, private _translate: TranslateService, private _toast: ToastService,
                private _routeActivated: ActivatedRoute, private _router: Router) {
        this.project = this._routeActivated.snapshot.data['project'];
        this._pipStore.getPipelineType().subscribe(list => {
            this.ready = true;
            this.pipelineType = list.toArray();
            this.newPipeline.type = this.pipelineType[0];
        });

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
        this._pipStore.createPipeline(this.project.key, this.newPipeline).subscribe(() => {
            this.loadingCreate = false;
            this._toast.success('', this._translate.instant('pipeline_added'));
            this._router.navigate(['/project', this.project.key, 'pipeline', this.newPipeline.name]);
        }, () => {
            this.loadingCreate = false;
        });

    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], {queryParams: {tab: 'pipelines'}});
    }

    importPipeline() {
        this.loadingCreate = true;
        this._pipStore.importPipeline(this.project.key, this.pipToImport)
            .pipe(finalize(() => this.loadingCreate = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('pipeline_added'));
                this.goToProject();
            });
    }

    fileEvent(event) {
        this.pipToImport = event;
    }
}
