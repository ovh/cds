import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { WorkflowRunResult, WorkflowRunResultType } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";

export class WorkflowRunResultView {
	result: WorkflowRunResult;
	downloadLink: string;
	viewLink: string;
}

@Component({
	selector: 'app-run-results',
	templateUrl: './run-results.html',
	styleUrls: ['./run-results.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunResultsComponent implements OnChanges {
	@Input() results: Array<WorkflowRunResult>;
	@Output() onSelectResult = new EventEmitter<string>();

	filteredResults: Array<WorkflowRunResultView>;
	searchValue = '';
	filterModified: boolean;
	filtered: boolean;
	activeFilters: Array<string>;
	filterOptions = [];
	nodes = [];

	constructor(
		private _cd: ChangeDetectorRef
	) {
		this.filterOptions = Object.values(WorkflowRunResultType).map(t => ({ label: t, value: t, checked: t !== WorkflowRunResultType.tests })).sort();
		this.filtered = true;
		this.activeFilters = this.filterOptions.filter(o => o.checked).map(o => o.value);
		this.initResults();
	}

	ngOnChanges(changes: SimpleChanges): void {
		if (!this.filterModified) {
			const types = [...new Set((this.results ?? []).map(r => r.type).concat(...Object.values(WorkflowRunResultType)))].sort();
			this.filterOptions = types.map(t => ({ label: t, value: t, checked: t !== WorkflowRunResultType.tests }));
			this.filtered = true;
			this.activeFilters = this.filterOptions.filter(o => o.checked).map(o => o.value);
		}
		this.initResults();
		this._cd.markForCheck();
	}

	updateSearch(value: string): void {
		this.searchValue = value;
		this.initResults();
	}

	updateFilters(event): void {
		this.filterModified = true;
		this.filterOptions = event;
		this.filtered = !this.filterOptions.map(o => o.checked).reduce((p, c) => p && c);
		this.activeFilters = this.filterOptions.filter(o => o.checked).map(o => o.value);
		this.initResults();
		this._cd.markForCheck();
	}

	clickResult(r: WorkflowRunResult): void {
		this.onSelectResult.emit(r.id);
	}

	initResults(): void {
		this.filteredResults = (this.results ?? []).filter(r => {
			const typeMatch = this.activeFilters.indexOf(r.type) !== -1
			const identiferMatch = !this.searchValue || r.identifier.toLowerCase().indexOf(this.searchValue.toLowerCase()) !== -1;
			return typeMatch && identiferMatch;
		}).map(r => {
			return {
				result: r,
				downloadLink: this.generateDownloadLink(r),
				viewLink: this.generateViewLink(r)
			};
		});
	}

	generateDownloadLink(result: WorkflowRunResult): string {
		let downloadLink = result.artifact_manager_metadata['downloadURI'];
		if (downloadLink) {
			return downloadLink;
		}
		const downloadPath = result.artifact_manager_metadata['cdn_download_path'];
		if (downloadPath) {
			return './cdscdn' + downloadPath;
		}
		return null;
	}

	generateViewLink(result: WorkflowRunResult): string {
		return result.detail.data['uri'];
	}

}