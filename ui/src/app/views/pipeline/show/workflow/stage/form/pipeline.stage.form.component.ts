import { Component, Input, OnInit } from '@angular/core';
import { WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { finalize } from 'rxjs/operators';
import { PermissionValue } from '../../../../../../model/permission.model';
import { Pipeline } from '../../../../../../model/pipeline.model';
import { Project } from '../../../../../../model/project.model';
import { Stage } from '../../../../../../model/stage.model';

@Component({
    selector: 'app-pipeline-stage-form',
    templateUrl: './pipeline.stage.form.html',
    styleUrls: ['./pipeline.stage.form.scss']
})
export class PipelineStageFormComponent implements OnInit {

    @Input() project: Project;
    @Input() pipeline: Pipeline;
    @Input() stage: Stage;

    permissionEnum = PermissionValue;
    triggerConditions: WorkflowTriggerConditionCache;
    loading = true;

    constructor(private pipelineService: PipelineService) {

    }

    ngOnInit(): void {
        this.pipelineService.getStageConditionsName(this.project.key, this.pipeline.name)
            .pipe(finalize(() => this.loading = false))
            .subscribe((conditions) => this.triggerConditions = conditions);
    }
}
