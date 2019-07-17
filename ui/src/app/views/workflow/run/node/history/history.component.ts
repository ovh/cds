import { ChangeDetectionStrategy, Component, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { Parameter } from 'app/model/parameter.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { Column, ColumnType } from 'app/shared/table/data-table.component';

@Component({
    selector: 'app-workflow-node-run-history',
    templateUrl: './history.html',
    styleUrls: ['./history.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowNodeRunHistoryComponent implements OnInit {
    @Input() project: Project;
    @Input() run: WorkflowRun;
    @Input() history: Array<WorkflowNodeRun>;
    @Input() currentBuild: WorkflowNodeRun;
    @Input() workflowName: string;

    loading: boolean;
    columns: Array<Column<WorkflowNodeRun>>;

    constructor(private _router: Router) {

    }

    ngOnInit() {
        this.columns = [
            <Column<WorkflowNodeRun>>{
                type: ColumnType.ICON,
                name: 'common_status',
                selector: (nodeRun: WorkflowNodeRun) => {
                    if (nodeRun.status === PipelineStatus.FAIL || nodeRun.status === PipelineStatus.STOPPED) {
                        return ['remove', 'red', 'icon'];
                    }
                    if (nodeRun.status === PipelineStatus.SUCCESS) {
                        return ['check', 'green', 'icon'];
                    }
                    if (nodeRun.status === PipelineStatus.WAITING || nodeRun.status === PipelineStatus.BUILDING) {
                        return ['wait', 'blue', 'icon'];
                    }
                    if (PipelineStatus.neverRun(nodeRun.status)) {
                        return ['ban', 'grey', 'icon'];
                    }
                    return ['stop', 'grey', 'icon'];
                }
            },
            <Column<WorkflowNodeRun>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_version',
                selector: (nodeRun: WorkflowNodeRun) => {
                    let node = Workflow.getNodeByID(nodeRun.workflow_node_id, this.run.workflow);
                    let url = this._router.createUrlTree([
                        '/project',
                        this.project.key,
                        'workflow',
                        this.workflowName,
                        'run',
                        nodeRun.num,
                        'node',
                        nodeRun.id
                    ], { queryParams: { sub: nodeRun.subnumber, name: Workflow.getPipeline(this.run.workflow, node).name } });
                    return {
                        link: url.toString(),
                        value: `${nodeRun.num}.${nodeRun.subnumber}`
                    };
                }
            },
            <Column<WorkflowNodeRun>>{
                type: ColumnType.TEXT,
                name: 'common_trigger_by',
                selector: (nr: WorkflowNodeRun) => {
                    if (nr.build_parameters) {
                        let userParam: Parameter;
                        userParam = nr.build_parameters.find(p => p.name === 'cds.triggered_by.username');
                        if (userParam) {
                            return userParam.value;
                        }
                        userParam = nr.build_parameters.find(p => p.name === 'git.author');
                        if (userParam) {
                            return userParam.value;
                        }
                    }
                }
            },
            <Column<WorkflowNodeRun>>{
                type: ColumnType.DATE,
                name: 'common_date_start',
                selector: (nr: WorkflowNodeRun) => nr.start,
            },
            <Column<WorkflowNodeRun>>{
                type: ColumnType.DATE,
                name: 'common_date_end',
                selector: (nr: WorkflowNodeRun) => nr.done,
            }
        ];
    }

    currentSelect(): (nodeRun: WorkflowNodeRun) => boolean {
        return (nodeRun: WorkflowNodeRun) => {
            if (!this.currentBuild || !nodeRun) {
                return false;
            }
            return nodeRun.id === this.currentBuild.id && nodeRun.subnumber === this.currentBuild.subnumber;
        };
    }

    goToSubNumber(nodeRun: WorkflowNodeRun): void {
        let node = Workflow.getNodeByID(nodeRun.workflow_node_id, this.run.workflow);
        this._router.navigate(['/project', this.project.key, 'workflow', this.workflowName, 'run', nodeRun.num, 'node',
            nodeRun.id], { queryParams: { sub: nodeRun.subnumber, name: Workflow.getPipeline(this.run.workflow, node).name } });
    }
}
