import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { TestCase } from "app/model/pipeline.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { PreferencesState } from "app/store/preferences.state";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Subscription } from "rxjs";

@Component({
	selector: 'app-run-test',
	templateUrl: './run-test.html',
	styleUrls: ['./run-test.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunTestComponent implements OnInit, OnChanges, OnDestroy {
	@ViewChild('editor') editor: NzCodeEditorComponent;

	@Input() test: TestCase;
	@Output() onClose = new EventEmitter<void>();

	editorOption: EditorOptions;
	resizingSubscription: Subscription;
	tabs: Array<Tab>;
	selectedTab: Tab;
	testRaw: string;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) {
		this.tabs = [<Tab>{
			title: 'Output',
			key: 'output',
			default: true
		}];
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.editorOption = {
			language: 'json',
			minimap: { enabled: false },
			readOnly: true,
			wordWrap: 'on'
		};

		this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
			if (!resizing && this.editor) {
				this.editor.layout();
			}
		});
	}

	ngOnChanges(): void {
		this.testRaw = JSON.stringify(this.test, null, 2);
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