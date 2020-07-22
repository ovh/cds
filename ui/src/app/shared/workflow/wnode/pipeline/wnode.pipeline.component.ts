import { ChangeDetectionStrategy, Component, Input, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';

@Component({
    selector: 'app-workflow-wnode-pipeline',
    templateUrl: './node.pipeline.html',
    styleUrls: ['./node.pipeline.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodePipelineComponent implements OnDestroy {
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public warnings: number;

    project: Project;
    selected: boolean;
    pipelineStatus = PipelineStatus;

    constructor(
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _store: Store
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    displayLogs() {
        if (this.noderun) {
            this._router.navigate(['node', this.noderun.id], {
                relativeTo: this._activatedRoute, queryParams: { name: this.node.name }
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
