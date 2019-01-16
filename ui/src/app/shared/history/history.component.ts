import { Component, EventEmitter, Input, Output } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { PipelineBuild } from '../../model/pipeline.model';
import { Project } from '../../model/project.model';
import { ApplicationPipelineService } from '../../service/application/pipeline/application.pipeline.service';
import { Table } from '../table/table';
import { ToastService } from '../toast/ToastService';

@Component({
    selector: 'app-history',
    templateUrl: './history.html',
    styleUrls: ['./history.scss']
})
export class HistoryComponent extends Table<PipelineBuild> {
    @Input() project: Project;
    @Input() history: Array<PipelineBuild>;
    @Input() currentBuild: PipelineBuild;
    @Output() buildDeletedEvent = new EventEmitter<boolean>();

    loading: boolean;

    constructor(
        private _appBuildSerivce: ApplicationPipelineService,
        private _translate: TranslateService,
        private _toast: ToastService
    ) {
        super();
    }

    getData(): Array<PipelineBuild> {
        return this.history;
    }

    getTriggerSource(pb: PipelineBuild): string {
        return PipelineBuild.GetTriggerSource(pb);
    }

    deleteBuild(pb: PipelineBuild): void {
        this.loading = true;
        this._appBuildSerivce.deleteBuild(this.project.key, pb.application.name, pb.pipeline.name,
            pb.environment.name, pb.build_number)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.buildDeletedEvent.emit(true);
                this._toast.success('', this._translate.instant('pipeline_build_deleted'));
            });
    }
}

