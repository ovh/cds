import {Component, Input} from '@angular/core';
import {Table} from '../../../../../shared/table/table';
import {Project} from '../../../../../model/project.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../../../model/workflow.run.model';
import {Parameter} from '../../../../../model/parameter.model';

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

    loading: boolean;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.history;
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

