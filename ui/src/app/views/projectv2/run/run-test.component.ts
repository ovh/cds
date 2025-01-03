import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { TestCase, Tests } from "app/model/pipeline.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { PreferencesState } from "app/store/preferences.state";
import { editor } from "monaco-editor";
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

	@Input() tests: Tests;
	@Input() test: string;

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
		let t: TestCase;
		for (let i = 0; i < this.tests.test_suites.length; i++) {
			for (let j = 0; j < this.tests.test_suites[i].tests.length; j++) {
				const key = this.tests.test_suites[i].name + '/' + this.tests.test_suites[i].tests[j].name;
				if (key === this.test) {
					t = this.tests.test_suites[i].tests[j];
					break
				}
			}
		}
		this.testRaw = JSON.stringify(t, null, 2);
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