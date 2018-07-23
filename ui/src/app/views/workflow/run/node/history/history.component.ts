import {Component, Input} from '@angular/core';
import {Router} from '@angular/router';
import {Parameter} from '../../../../../model/parameter.model';
import {Project} from '../../../../../model/project.model';
import {Workflow} from '../../../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../../../model/workflow.run.model';
import {Table} from '../../../../../shared/table/table';

@Component({
    selector: 'app-workflow-node-run-history',
    templateUrl: './history.html',
    styleUrls: ['./history.scss']
})
export class WorkflowNodeRunHistoryComponent extends Table {

    @Input() project: Project;
    @Input() run: WorkflowRun;
    @Input() history: Array<WorkflowNodeRun>;
    @Input() currentBuild: WorkflowNodeRun;
    @Input() workflowName: string;

    loading: boolean;

    constructor(private _router: Router) {
        super();
    }

    getData(): any[] {
        return this.history;
    }

    goToSubNumber(nodeRun: WorkflowNodeRun): void {
        let node = Workflow.getNodeByID(nodeRun.workflow_node_id, this.run.workflow);
        this._router.navigate(['/project', this.project.key, 'workflow', this.workflowName, 'run', nodeRun.num, 'node',
            nodeRun.id], {queryParams: {sub: nodeRun.subnumber, name: node.pipeline_name}});
    }

    getTriggerSource(nr: WorkflowNodeRun): string {
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
}
