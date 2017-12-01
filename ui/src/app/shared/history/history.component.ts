import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Table} from '../table/table';
import {PipelineBuild} from '../../model/pipeline.model';
import {Project} from '../../model/project.model';
import {ApplicationPipelineService} from '../../service/application/pipeline/application.pipeline.service';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../toast/ToastService';

@Component({
    selector: 'app-history',
    templateUrl: './history.html',
    styleUrls: ['./history.scss']
})
export class HistoryComponent extends Table {

    @Input() project: Project;
    @Input() history: Array<PipelineBuild>;
    @Input() currentBuild: PipelineBuild;
    @Output() buildDeletedEvent = new EventEmitter<boolean>();

    loading: boolean;

    constructor(private _appBuildSerivce: ApplicationPipelineService, private _translate: TranslateService, private _toast: ToastService) {
        super();
    }

    getData(): any[] {
        return this.history;
    }

    getTriggerSource(pb: PipelineBuild): string {
        return PipelineBuild.GetTriggerSource(pb);
    }

    deleteBuild(pb: PipelineBuild): void {
        this.loading = true;
        this._appBuildSerivce.deleteBuild(
            this.project.key, pb.application.name, pb.pipeline.name, pb.environment.name, pb.build_number).subscribe(() => {
           this._toast.success('', this._translate.instant('pipeline_build_deleted'));
           this.loading = false;
           this.buildDeletedEvent.emit(true);
        }, () => {
            this.loading = false;
        });
    }
}

