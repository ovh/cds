import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {Router} from '@angular/router';
import {WorkflowNodeRun, WorkflowRunRequest} from '../../../../../model/workflow.run.model';
import {Project} from '../../../../../model/project.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {Workflow, WorkflowNode} from '../../../../../model/workflow.model';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {WorkflowNodeRunParamComponent} from '../../../../../shared/workflow/node/run/node.run.param.component';

@Component({
    selector: 'app-workflow-node-run-summary',
    templateUrl: './run.summary.html',
    styleUrls: ['./run.summary.scss']
})
export class WorkflowNodeRunSummaryComponent implements OnInit {

    @Input() nodeRun: WorkflowNodeRun;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() duration: string;

    @ViewChild('workflowNodeRunParam')
    runWithParamComponent: WorkflowNodeRunParamComponent;

    node: WorkflowNode;
    pipelineStatusEnum = PipelineStatus;

    loading = false;

    constructor(private _router: Router, private _wrService: WorkflowRunService, private _toast: ToastService,
                private _translate: TranslateService) {
    }

    ngOnInit(): void {
        this.node = Workflow.getNodeByID(this.nodeRun.workflow_node_id, this.workflow);
    }

    getName(): string {
        if (this.nodeRun && this.node) {
            return this.node.pipeline.name + ' ' + this.nodeRun.num + '.' + this.nodeRun.subnumber;
        }
    }

    getAuthor(): string {
        if (this.nodeRun) {
            return '';
        }

    }

    navigateToBuild(nr: WorkflowNodeRun): void {

    }

    stop(): void {
        this.loading = true;
        this._wrService.stopNodeRun(this.project.key, this.workflow.name, this.nodeRun.num, this.nodeRun.id)
            .finally(() => this.loading = false)
            .first().subscribe(() => {
            this._toast.success('', this._translate.instant('pipeline_stop'));
        });
    }

    runNew(): void {
        let request = new WorkflowRunRequest();
        request.from_node = this.nodeRun.workflow_node_id;
        request.number = this.nodeRun.num;
        request.manual = this.nodeRun.manual;
        request.hook = this.nodeRun.hook_event;
        this._wrService.runWorkflow(this.project.key, this.workflow, request).subscribe(wr => {
            this._router.navigate(['project', this.project.key, 'workflow', this.workflow.name, 'run', this.nodeRun.num]);
        });

    }

    runNewWithParameter(): void {
        if (this.runWithParamComponent) {
            this.runWithParamComponent.show();
        }
    }
}
