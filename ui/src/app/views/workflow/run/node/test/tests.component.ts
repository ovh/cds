import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { Coverage } from 'app/model/coverage.model';
import { Tests } from 'app/model/pipeline.model';

@Component({
    selector: 'app-workflow-tests-result',
    templateUrl: './tests.result.html',
    styleUrls: ['./tests.result.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowRunTestsResultComponent {

    @Input() tests: Tests;

    _coverage: Coverage;
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
    ) {
    }

    @Input('coverage')
    set coverage(data: Coverage) {
        if (data && data.workflow_id) {
            this._coverage = data;
            if (this._coverage.report.total_branches) {
                this.percentBranches =
                    parseFloat((this._coverage.report.covered_branches * 100 / this._coverage.report.total_branches)
                    .toFixed(2));
                if (this._coverage.trend.current_branch_report.total_branches &&
                    this._coverage.trend.current_branch_report.total_branches > 0 ) {
                    this.currentBranchTrendBranches =
                        parseFloat((this._coverage.trend.current_branch_report.covered_branches * 100
                        / this._coverage.trend.current_branch_report.total_branches).toFixed(2));
                }
                if (this._coverage.trend.default_branch_report.total_branches &&
                    this._coverage.trend.default_branch_report.total_branches > 0) {
                    this.defaultBranchTrendBranches =
                        parseFloat((this._coverage.trend.default_branch_report.covered_branches * 100
                        / this._coverage.trend.default_branch_report.total_branches).toFixed(2));
                }
            }
            if (this._coverage.report.total_functions) {
                this.percentFunctions =
                    parseFloat((this._coverage.report.covered_functions * 100 / this._coverage.report.total_functions)
                        .toFixed(2));
                if (this._coverage.trend.current_branch_report.total_functions &&
                    this._coverage.trend.current_branch_report.total_functions > 0) {
                    this.currentBranchTrendFunctions =
                        parseFloat((this._coverage.trend.current_branch_report.covered_functions * 100
                        / this._coverage.trend.current_branch_report.total_functions).toFixed(2));
                }
                if (this._coverage.trend.default_branch_report.total_branches &&
                    this._coverage.trend.default_branch_report.total_branches > 0) {
                    this.defaultBranchTrendFunctions =
                        parseFloat((this._coverage.trend.default_branch_report.covered_functions * 100
                        / this._coverage.trend.default_branch_report.total_branches).toFixed(2));
                }
            }
            if (this._coverage.report.total_lines && this._coverage.report.total_lines > 0) {
                this.percentLines =
                    parseFloat((this._coverage.report.covered_lines * 100 / this._coverage.report.total_lines).toFixed(2));
                if (this._coverage.trend.current_branch_report.total_lines && this._coverage.trend.current_branch_report.total_lines > 0) {
                    this.currentBranchTrendLines =
                        parseFloat((this._coverage.trend.current_branch_report.covered_lines * 100
                        / this._coverage.trend.current_branch_report.total_lines).toFixed(2));
                }
                if (this._coverage.trend.default_branch_report.total_lines && this._coverage.trend.default_branch_report.total_lines > 0) {
                    this.defaultBranchTrendLines =
                        parseFloat((this._coverage.trend.default_branch_report.covered_lines * 100
                        / this._coverage.trend.default_branch_report.total_lines).toFixed(2));
                }
            }
        }
        this._cd.detectChanges();
    };
    get coverage(): Coverage {
        return this._coverage;
    }
}
