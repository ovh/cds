import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { finalize } from 'rxjs/operators';
import { Pipeline } from '../../../../../../model/pipeline.model';
import { Project } from '../../../../../../model/project.model';
import { Stage } from '../../../../../../model/stage.model';

@Component({
    selector: 'app-pipeline-stage-form',
    templateUrl: './pipeline.stage.form.html',
    styleUrls: ['./pipeline.stage.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class PipelineStageFormComponent implements OnInit {

    @Input() project: Project;
    @Input() pipeline: Pipeline;
    @Input() stage: Stage;
    @Input() readOnly: boolean;

    triggerConditions: WorkflowTriggerConditionCache;
    loading: boolean;

    constructor(
        private pipelineService: PipelineService,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        this.loading = true;
        this._cd.markForCheck();
        this.pipelineService.getStageConditionsName(this.project.key, this.pipeline.name)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((conditions) => this.triggerConditions = conditions);
    }
}
