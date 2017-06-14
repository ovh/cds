import {Component, Input} from '@angular/core';
import {WorkflowNodeJobRun, WorkflowNodeRun} from '../../../../../model/workflow.run.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';

@Component({
    selector: 'app-node-run-pipeline',
    templateUrl: './pipeline.html',
    styleUrls: ['./pipeline.scss']
})
export class WorkflowRunNodePipelineComponent {

    nodeRun: WorkflowNodeRun;

    @Input('run')
    set run(data: WorkflowNodeRun) {
         this.nodeRun = data;
    }

    pipelineStatusEnum = PipelineStatus;
    selectedRunJob: WorkflowNodeJobRun;

    constructor() { }

    selectedJob(rj: WorkflowNodeJobRun): void {
        this.selectedRunJob = rj;
    }
}
