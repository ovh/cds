import { Component, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { DeletePipeline, UpdatePipeline } from 'app/store/pipelines.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize } from 'rxjs/operators';
import { Pipeline } from '../../../../model/pipeline.model';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-pipeline-admin',
    templateUrl: './pipeline.admin.html',
    styleUrls: ['./pipeline.admin.scss']
})
export class PipelineAdminComponent implements OnInit {

    public loading = false;

    editablePipeline: Pipeline;
    oldName: string;

    @Input() project;

    @Input('pipeline')
    set pipeline(data: Pipeline) {
        this.oldName = data.name;
        this.editablePipeline = cloneDeep(data);
    }

    constructor(
        private store: Store,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router
    ) {

    }

    ngOnInit(): void {
        if (!this.project.permissions.writable) {
            this._router.navigate([
                '/project',
                this.project.key,
                'pipeline',
                this.editablePipeline.name
            ], { queryParams: { tab: 'pipeline' } });
        }
    }

    updatePipeline(): void {
        this.loading = true;
        this.store.dispatch(new UpdatePipeline({
            projectKey: this.project.key,
            pipelineName: this.oldName,
            changes: this.editablePipeline
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('pipeline_updated'));
                if (this.oldName !== this.editablePipeline.name) {
                    this._router.navigate(
                        ['project', this.project.key, 'pipeline', this.editablePipeline.name],
                        { queryParams: { 'tab': 'advanced' } }
                    );
                }
            });
    }

    deletePipeline(): void {
        this.loading = true;
        this.store.dispatch(new DeletePipeline({
            projectKey: this.project.key,
            pipelineName: this.editablePipeline.name
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('pipeline_deleted'));
                this._router.navigate(
                    ['project', this.project.key],
                    { queryParams: { 'tab': 'pipelines' } }
                );
            });
    }
}
