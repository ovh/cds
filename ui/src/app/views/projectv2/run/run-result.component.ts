import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, SimpleChanges, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { V2WorkflowRun, WorkflowRunResult } from "app/model/v2.workflow.run.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { PreferencesState } from "app/store/preferences.state";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Subscription } from "rxjs";

@Component({
	selector: 'app-run-result',
	templateUrl: './run-result.html',
	styleUrls: ['./run-result.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunResultComponent implements OnInit, OnChanges, OnDestroy {
	@ViewChild('editor') editor: NzCodeEditorComponent;

	@Input() run: V2WorkflowRun;
	@Input() resultID: string;
	@Output() onClickClose = new EventEmitter<void>();

	editorOption: EditorOptions;
	resizingSubscription: Subscription;
	defaultTabs: Array<Tab>;
	tabs: Array<Tab>;
	selectedTab: Tab;
	result: WorkflowRunResult;
	resultRaw: string;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) {
		this.defaultTabs = [<Tab>{
			title: 'Raw',
			key: 'raw'
		}];
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.editorOption = {
			language: 'json',
			minimap: { enabled: false },
			readOnly: true
		};

		this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
			if (!resizing && this.editor) {
				this.editor.layout();
			}
		});
	}

	ngOnChanges(): void {
		this.result = this.run.results.find(r => r.id === this.resultID);
		if (this.result.type === 'tests') {
			this.tabs = [<Tab>{
				title: 'Tests',
				key: 'tests',
			}, ...this.defaultTabs];
		} else {
			this.tabs = [...this.defaultTabs];
		}
		this.tabs[0].default = true;
		this.resultRaw = JSON.stringify(this.result, null, 2);
		this._cd.markForCheck();
	}

	selectTab(tab: Tab): void {
		this.selectedTab = tab;
		this._cd.markForCheck();
	}

	clickClose(): void {
		this.onClickClose.emit();
	}

}