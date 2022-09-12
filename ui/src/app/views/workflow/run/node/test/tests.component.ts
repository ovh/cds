import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Select } from '@ngxs/store';
import { Tests } from 'app/model/pipeline.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-tests-result',
    templateUrl: './tests.result.html',
    styleUrls: ['./tests.result.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunTestsResultComponent implements OnInit, OnDestroy {

    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;

    tests: Tests;

    percentLines: number;
    currentBranchTrendLines: number;
    defaultBranchTrendLines: number;
    percentFunctions: number;
    currentBranchTrendFunctions: number;
    defaultBranchTrendFunctions: number;
    percentBranches: number;
    currentBranchTrendBranches: number;
    defaultBranchTrendBranches: number;

    constructor(
        private _cd: ChangeDetectorRef
    ) {    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
            if (!nr || (!nr.tests)) {
                return;
            }
            if ( (!this.tests && nr.tests) || (this.tests && nr.tests && this.tests.total !== nr.tests.total)) {
                this.tests = nr.tests;
                this._cd.markForCheck();
            }
        });
    }
}
