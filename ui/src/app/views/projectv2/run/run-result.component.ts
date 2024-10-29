import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { PreferencesState } from "app/store/preferences.state";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Subscription } from "rxjs";
import { WorkflowRunResult } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { editor } from "monaco-editor";

@Component({
	selector: 'app-run-result',
	templateUrl: './run-result.html',
	styleUrls: ['./run-result.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunResultComponent implements OnInit, OnChanges, OnDestroy {
	@ViewChild('editor') editor: NzCodeEditorComponent;

	@Input() result: WorkflowRunResult;

	editorOption: EditorOptions;
	resizingSubscription: Subscription;
	tabs: Array<Tab>;
	selectedTab: Tab;
	resultRaw: string;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) {
		this.tabs = [<Tab>{
			title: 'Description',
			key: 'description',
			default: true
		}, <Tab>{
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
		this.resultRaw = JSON.stringify(this.result, null, 2);
		this._cd.markForCheck();
	}

	selectTab(tab: Tab): void {
		this.selectedTab = tab;
		this._cd.markForCheck();
	}

	onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
		this.editor.layout();
	}

}