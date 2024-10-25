import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { Tests } from 'app/model/pipeline.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowState } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-tests-result',
    templateUrl: './tests.result.html',
    styleUrls: ['./tests.result.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunTestsResultComponent implements OnInit, OnDestroy {

    nodeRunSubs: Subscription;

    tests: Tests;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) {    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.nodeRunSubs = this._store.select(WorkflowState.getSelectedNodeRun()).subscribe(nr => {
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
