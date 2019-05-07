import { Component, Input, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import {PipelineStatus} from '@cds/model/pipeline.model';
import {Project} from '@cds/model/project.model';
import {WNode, Workflow} from '@cds/model/workflow.model';
import {WorkflowNodeRun} from '@cds/model/workflow.run.model';
import {AutoUnsubscribe} from '@cds/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode-pipeline',
    templateUrl: './node.pipeline.html',
    styleUrls: ['./node.pipeline.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodePipelineComponent implements OnInit {
    @Input() public project: Project;
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public warnings: number;
    selected: boolean;
    pipelineStatus = PipelineStatus;

    constructor(
        private _activatedRoute: ActivatedRoute,
        private _router: Router
    ) { }

    ngOnInit(): void {
    }

    displayLogs() {
        if (this.noderun) {
            this._router.navigate(['node', this.noderun.id], {
                relativeTo: this._activatedRoute,
            });
        } else {
            this._router.navigate([
                '/project', this.project.key,
                'pipeline', Workflow.getPipeline(this.workflow, this.node).name
            ], {
                    queryParams: {
                        workflow: this.workflow.name
                    }
                }
            );
        }
    }
}
