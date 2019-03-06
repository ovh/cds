import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { FetchAsCodePipeline, ImportPipeline, PreviewPipeline, ResyncPipeline } from 'app/store/pipelines.action';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Pipeline, PipelineStatus } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { PipelineCoreService } from '../../../../service/pipeline/pipeline.core.service';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-pipeline-ascode-editor',
    templateUrl: './pipeline.ascode.editor.html',
    styleUrls: ['./pipeline.ascode.editor.scss']
})
@AutoUnsubscribe()
export class PipelineAsCodeEditorComponent {

    // Project that contains the pipeline
    @Input() project: Project;
    @Input() pipeline: Pipeline;
    // Flag indicate if sidebar is open
    @Input('open')
    set open(data: boolean) {
        if (data && !this.updated) {
            this.loadingGet = true;
            this.store.dispatch(new FetchAsCodePipeline({
                projectKey: this.project.key,
                pipelineName: this.pipeline.name
            })).pipe(finalize(() => this.loadingGet = false))
                .subscribe(() => this.exportedPip = this.pipeline.asCode);
        }
        this._open = data;
    }
    get open() {
        return this._open;
    }
    _open = false;

    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    asCodeEditorSubscription: Subscription;
    codeMirrorConfig: any;

    updated = false;
    loading = false;
    loadingGet = true;
    previewMode = false;
    exportedPip = '';
    statusEnum = PipelineStatus;

    constructor(
        private store: Store,
        private _pipCoreService: PipelineCoreService,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };

        this.asCodeEditorSubscription = this._pipCoreService.getAsCodeEditor()
            .subscribe((state) => {
                if (state != null && state.save) {
                    this.save();
                }
            });
    }

    keyEvent(event: KeyboardEvent) {
        if (event.key === 's' && (event.ctrlKey || event.metaKey)) {
            this.save();
            event.preventDefault();
        }
    }

    cancel() {
        if (this.previewMode) {
            this.store.dispatch(new ResyncPipeline({
                projectKey: this.project.key,
                pipelineName: this.pipeline.name
            })).subscribe(() => this._pipCoreService.toggleAsCodeEditor({ open: false, save: false }));
            this.previewMode = false;
        } else {
            this._pipCoreService.toggleAsCodeEditor({ open: false, save: false });
        }
    }

    preview() {
        this.loading = true;
        this.previewMode = true;
        this.store.dispatch(new PreviewPipeline({
            projectKey: this.project.key,
            pipelineName: this.pipeline.name,
            pipCode: this.exportedPip
        })).pipe(finalize(() => this.loading = false))
            .subscribe();
    }

    save() {
        this.loading = true;
        this.store.dispatch(new ImportPipeline({
            projectKey: this.project.key,
            pipName: this.pipeline.name,
            pipelineCode: this.exportedPip
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._pipCoreService.toggleAsCodeEditor({ open: false, save: false });
                this._pipCoreService.setPipelinePreview(null);
                this._toast.success('', this._translate.instant('pipeline_updated'));
            });
    }
}
