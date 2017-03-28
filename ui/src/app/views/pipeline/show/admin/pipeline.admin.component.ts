import {Component, Input, OnInit} from '@angular/core';
import {Pipeline} from '../../../../model/pipeline.model';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {Router} from '@angular/router';

declare var _: any;

@Component({
    selector: 'app-pipeline-admin',
    templateUrl: './pipeline.admin.html',
    styleUrls: ['./pipeline.admin.scss']
})
export class PipelineAdminComponent implements OnInit {

    public loading = false;

    editablePipeline: Pipeline;
    pipelineTypes: Array<string>;
    oldName: string;

    @Input() project;

    @Input('pipeline')
    set pipeline(data: Pipeline) {
        this.oldName = data.name;
        this.editablePipeline = _.cloneDeep(data);
    }

    constructor(private _pipStore: PipelineStore, private _toast: ToastService, private _translate: TranslateService,
                private _router: Router) {
        this._pipStore.getPipelineType().subscribe( types => {
            this.pipelineTypes = types.toArray();
        });
    }

    ngOnInit(): void {
        if (this.editablePipeline.permission !== 7) {
            this._router.navigate(['/project', this.project.key, 'pipeline', this.editablePipeline.name], {queryParams: {tab: 'pipeline'}});
        }
    }

    updatePipeline(): void {
        this.loading = true;
        this._pipStore.updatePipeline(this.project.key, this.oldName, this.editablePipeline).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('pipeline_updated'));
            this._router.navigate(
                ['project', this.project.key, 'pipeline', this.editablePipeline.name],
                { queryParams: { 'tab' : 'advanced' }}
            );
        }, () => {
            this.loading = false;
        });
    }

    deletePipeline(): void {
        this.loading = true;
        this._pipStore.deletePipeline(this.project.key, this.editablePipeline.name).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('pipeline_deleted'));
            this._router.navigate(
                ['project', this.project.key],
                { queryParams: { 'tab' : 'pipelines' }}
            );
        }, () => {
            this.loading = false;
        });
    }
}
