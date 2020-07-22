import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Select } from '@ngxs/store';
import { Coverage } from 'app/model/coverage.model';
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
    coverage: Coverage;

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
            if (!nr || (!nr.tests && !nr.coverage)) {
                return;
            }
            if ( (!this.tests && nr.tests) || (this.tests && nr.tests && this.tests.total !== nr.tests.total)) {
                this.tests = nr.tests;
                this._cd.markForCheck();
            }
            if (!this.coverage && nr.coverage) {
                this.coverage = nr.coverage;
                if (this.coverage.report.total_branches) {
                    this.percentBranches =
                        parseFloat((this.coverage.report.covered_branches * 100 / this.coverage.report.total_branches)
                            .toFixed(2));
                    if (this.coverage.trend.current_branch_report.total_branches &&
                        this.coverage.trend.current_branch_report.total_branches > 0 ) {
                        this.currentBranchTrendBranches =
                            parseFloat((this.coverage.trend.current_branch_report.covered_branches * 100
                                / this.coverage.trend.current_branch_report.total_branches).toFixed(2));
                    }
                    if (this.coverage.trend.default_branch_report.total_branches &&
                        this.coverage.trend.default_branch_report.total_branches > 0) {
                        this.defaultBranchTrendBranches =
                            parseFloat((this.coverage.trend.default_branch_report.covered_branches * 100
                                / this.coverage.trend.default_branch_report.total_branches).toFixed(2));
                    }
                }
                if (this.coverage.report.total_functions) {
                    this.percentFunctions =
                        parseFloat((this.coverage.report.covered_functions * 100 / this.coverage.report.total_functions)
                            .toFixed(2));
                    if (this.coverage.trend.current_branch_report.total_functions &&
                        this.coverage.trend.current_branch_report.total_functions > 0) {
                        this.currentBranchTrendFunctions =
                            parseFloat((this.coverage.trend.current_branch_report.covered_functions * 100
                                / this.coverage.trend.current_branch_report.total_functions).toFixed(2));
                    }
                    if (this.coverage.trend.default_branch_report.total_branches &&
                        this.coverage.trend.default_branch_report.total_branches > 0) {
                        this.defaultBranchTrendFunctions =
                            parseFloat((this.coverage.trend.default_branch_report.covered_functions * 100
                                / this.coverage.trend.default_branch_report.total_branches).toFixed(2));
                    }
                }
                if (this.coverage.report.total_lines && this.coverage.report.total_lines > 0) {
                    this.percentLines =
                        parseFloat((this.coverage.report.covered_lines * 100 / this.coverage.report.total_lines).toFixed(2));
                    if (this.coverage.trend.current_branch_report.total_lines &&
                        this.coverage.trend.current_branch_report.total_lines > 0) {
                        this.currentBranchTrendLines =
                            parseFloat((this.coverage.trend.current_branch_report.covered_lines * 100
                                / this.coverage.trend.current_branch_report.total_lines).toFixed(2));
                    }
                    if (this.coverage.trend.default_branch_report.total_lines &&
                        this.coverage.trend.default_branch_report.total_lines > 0) {
                        this.defaultBranchTrendLines =
                            parseFloat((this.coverage.trend.default_branch_report.covered_lines * 100
                                / this.coverage.trend.default_branch_report.total_lines).toFixed(2));
                    }
                }
                this._cd.markForCheck();
            }

        });
    }
}
