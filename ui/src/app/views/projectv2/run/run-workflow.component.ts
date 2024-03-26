import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { PreferencesState } from "app/store/preferences.state";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Subscription } from "rxjs";

@Component({
	selector: 'app-run-workflow',
	templateUrl: './run-workflow.html',
	styleUrls: ['./run-workflow.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunWorkflowComponent implements OnInit, OnChanges, OnDestroy {
	@ViewChild('editor') editor: NzCodeEditorComponent;

	@Input() workflow: string;
	@Output() onClose = new EventEmitter<void>();

	editorOption: EditorOptions;
	resizingSubscription: Subscription;
	defaultTabs: Array<Tab>;
	tabs: Array<Tab>;
	selectedTab: Tab;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) {
		this.defaultTabs = [<Tab>{
			title: 'Workflow',
			key: 'workflow'
		}];
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.editorOption = {
			language: 'yaml',
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
		this.tabs = [...this.defaultTabs];
		this.tabs[0].default = true;
		this._cd.markForCheck();
	}

	selectTab(tab: Tab): void {
		this.selectedTab = tab;
		this._cd.markForCheck();
	}

	clickClose(): void {
		this.onClose.emit();
	}

}