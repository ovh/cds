import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { finalize, first } from 'rxjs/operators';
import { PipelineStatus } from '../../../../../model/pipeline.model';
import { Project } from '../../../../../model/project.model';
import { WNode, Workflow } from '../../../../../model/workflow.model';
import { WorkflowNodeRun } from '../../../../../model/workflow.run.model';
import { WorkflowRunService } from '../../../../../service/workflow/run/workflow.run.service';
import { ToastService } from '../../../../../shared/toast/ToastService';
import { WorkflowNodeRunParamComponent } from '../../../../../shared/workflow/node/run/node.run.param.component';

@Component({
    selector: 'app-workflow-node-run-summary',
    templateUrl: './run.summary.html',
    styleUrls: ['./run.summary.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowNodeRunSummaryComponent implements OnInit {

    @Input() nodeRun: WorkflowNodeRun;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() duration: string;

    @ViewChild('workflowNodeRunParam', {static: false})
    runWithParamComponent: WorkflowNodeRunParamComponent;

    node: WNode;
    pipelineStatusEnum = PipelineStatus;

    loading = false;

    constructor(private _wrService: WorkflowRunService, private _toast: ToastService,
                private _translate: TranslateService, private _cd: ChangeDetectorRef) {
    }

    ngOnInit(): void {
        this.node = Workflow.getNodeByID(this.nodeRun.workflow_node_id, this.workflow);
    }

    stop(): void {
        this.loading = true;
        this._wrService.stopNodeRun(this.project.key, this.workflow.name, this.nodeRun.num, this.nodeRun.id)
            .pipe(
                first(),
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            ).subscribe(() => {
            this._toast.success('', this._translate.instant('pipeline_stop'));
        });
    }

    runNewWithParameter(): void {
        if (this.runWithParamComponent) {
            this.runWithParamComponent.show();
        }
    }
}
