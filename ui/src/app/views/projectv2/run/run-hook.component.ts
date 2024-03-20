import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { V2WorkflowRun } from "app/model/v2.workflow.run.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { PreferencesState } from "app/store/preferences.state";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Subscription } from "rxjs";

@Component({
	selector: 'app-run-hook',
	templateUrl: './run-hook.html',
	styleUrls: ['./run-hook.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunHookComponent implements OnInit, OnChanges, OnDestroy {
	@ViewChild('editor') editor: NzCodeEditorComponent;

	@Input() run: V2WorkflowRun;
	@Input() hook: string;
	@Output() onClickClose = new EventEmitter<void>();

	editorOption: EditorOptions;
	resizingSubscription: Subscription;
	event: string;
	tabs: Array<Tab>;
	selectedTab: Tab;

	constructor(
		private _store: Store,
		private _cd: ChangeDetectorRef
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.tabs = [<Tab>{
			title: 'Event',
			key: 'event',
			default: true
		}];

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
		if (this.run) {
			this.event = JSON.stringify(this.run.event, null, 2);
			this._cd.markForCheck();
		}
	}

	selectTab(tab: Tab): void {
		this.selectedTab = tab;
		this._cd.markForCheck();
	}

	clickClose(): void {
		this.onClickClose.emit();
	}

}