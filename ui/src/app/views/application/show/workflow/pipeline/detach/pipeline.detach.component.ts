import {Component, Input, OnInit} from '@angular/core';
import {Application} from '../../../../../../model/application.model';
import {WorkflowItem} from '../../../../../../model/application.workflow.model';
import {Pipeline} from '../../../../../../model/pipeline.model';

@Component({
    selector: 'app-application-pipeline-detach',
    templateUrl: './pipeline.detach.html',
    styleUrls: ['./pipeline.detach.scss']
})
export class ApplicationPipelineDetachComponent implements OnInit {

    @Input() application: Application;
    @Input() pipeline: Pipeline;

    workflowItems: Array<WorkflowItem>;

    constructor() {
        this.workflowItems = new Array<WorkflowItem>();
    }

    ngOnInit(): void {
        if (this.application.workflows) {
            this.checkTrees();
        }
    }

    checkTrees(): void {
        this.application.workflows.forEach(item => {
            this.addInTriggerList(item, false);
        });
    }


    addInTriggerList(item: WorkflowItem, sub: boolean): void {
        if (sub) {
            if (item.trigger.src_application.id === this.application.id && item.trigger.src_pipeline.id === this.pipeline.id) {
                this.workflowItems.push(item);
            } else if (item.trigger.dest_application.id === this.application.id && item.trigger.dest_pipeline.id === this.pipeline.id) {
                this.workflowItems.push(item);
            }
        }
        if (item.subPipelines) {
            item.subPipelines.forEach(sb => {
                this.addInTriggerList(sb, true);
            });
        }
    }

}
