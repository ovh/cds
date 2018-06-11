import {Component, Input, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {Project} from '../../../../model/project.model';
import {Pipeline} from '../../../../model/pipeline.model';
import {PipelineCoreService} from '../../../../service/pipeline/pipeline.core.service';
import {PipelineService} from '../../../../service/pipeline/pipeline.service';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {ToastService} from '../../../../shared/toast/ToastService';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/operators';

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
            this._pipelineService.getPipelineExport(this.project.key, this.pipeline.name)
                .pipe(finalize(() => this.loadingGet = false))
                .subscribe((wf) => this.exportedPip = wf);
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

    exportedPip: string;
    updated = false;
    loading = false;
    loadingGet = true;

    constructor(
        private _pipCoreService: PipelineCoreService,
        private _pipelineService: PipelineService,
        private _pipStore: PipelineStore,
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

    cancel() {
        this._pipCoreService.setPipelinePreview(null);
        this._pipCoreService.toggleAsCodeEditor({open: false, save: false});
    }

    preview() {
        this.loading = true;
        this._pipelineService.previewPipelineImport(this.project.key, this.exportedPip)
            .pipe(finalize(() => this.loading = false))
            .subscribe((pip) => this._pipCoreService.setPipelinePreview(pip));
    }

    save() {
        this.loading = true;
        this._pipStore.importPipeline(this.project.key, this.pipeline.name, this.exportedPip, true)
            .pipe(finalize(() => this.loading = false))
            .subscribe((pip) => {
                this._pipCoreService.toggleAsCodeEditor({open: false, save: false});
                this._pipCoreService.setPipelinePreview(null);
                this._toast.success('', this._translate.instant('pipeline_updated'));
            });
    }
}
