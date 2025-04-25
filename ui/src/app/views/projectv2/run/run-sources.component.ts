import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { PreferencesState } from "app/store/preferences.state";
import { editor } from "monaco-editor";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Subscription } from "rxjs";
import { V2WorkflowRun } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { dump } from "js-yaml";

@Component({
	selector: 'app-run-sources',
	templateUrl: './run-sources.html',
	styleUrls: ['./run-sources.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunSourcesComponent implements OnInit, OnChanges, OnDestroy {
	@ViewChild('editor') editor: NzCodeEditorComponent;

	@Input() run: V2WorkflowRun;

	editorOption: EditorOptions;
	resizingSubscription: Subscription;
	filenames: Array<string> = [];
	files: Array<string> = [];
	selectedFileIndex: number = 0;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) { }

	ngOnChanges(): void {
		let files = [dump(this.run.workflow_data.workflow, { lineWidth: -1 })];
		let filenames = ['workflow - ' + this.run.workflow_name];
		Object.keys(this.run.workflow_data.actions).sort().forEach((k) => {
			files.push(dump(this.run.workflow_data.actions[k], { lineWidth: -1 }));
			filenames.push('action - ' + k);
		});
		Object.keys(this.run.workflow_data.worker_models).sort().forEach((k) => {
			files.push(dump(this.run.workflow_data.worker_models[k], { lineWidth: -1 }));
			filenames.push('model - ' + k);
		});
		this.files = files;
		this.filenames = filenames;
		if (this.selectedFileIndex > this.files.length) { this.selectedFileIndex = 0; }
		this._cd.markForCheck();
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.editorOption = {
			language: 'yaml',
			minimap: { enabled: false },
			readOnly: true,
			scrollBeyondLastLine: false
		};

		this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
			if (!resizing && this.editor) {
				this.editor.layout();
			}
		});
	}

	selectFile(index: number): void {
		this.selectedFileIndex = index;
		this._cd.markForCheck();
	}

	onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
		this.editor.layout();
	}

}