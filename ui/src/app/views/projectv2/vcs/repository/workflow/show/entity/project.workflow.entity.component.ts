import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    ViewChild
} from "@angular/core";

import {editor,} from 'monaco-editor';

import {EditorOptions} from "ng-zorro-antd/code-editor/typings";
import {NzCodeEditorComponent} from "ng-zorro-antd/code-editor";
import {Store} from "@ngxs/store";
import {PreferencesState} from "app/store/preferences.state";
import * as actionPreferences from 'app/store/preferences.action';
import {Subscription} from 'rxjs';
import {Schema} from "app/model/json-schema.model";
import {AutoUnsubscribe} from "app/shared/decorator/autoUnsubscribe";
import {FlatSchema, JSONSchema} from "app/model/schema.model";
import Debounce from "app/shared/decorator/debounce";

declare const monaco: any;

@Component({
    selector: 'app-project-workflow-entity',
    templateUrl: './project.workflow.entity.html',
    styleUrls: ['./project.workflow.entity.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectWorkflowEntityComponent implements OnInit, OnChanges, OnDestroy {
    static PANEL_KEY = 'project-workflow-v2-entity-form';

    @ViewChild('editor') editor: NzCodeEditorComponent;

    @Input() path: string;
    @Input() data: string;
    @Input() schema: Schema;
    @Input() disabled: boolean;
    @Input() parentType: string;
    @Input() entityType: string;

    flatSchema: FlatSchema;
    dataGraph: string;
    dataEditor: string;
    editorOption: EditorOptions;
    panelSize: number | string;
    resizing: boolean;
    resizingSubscription: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) {
    }

    ngOnInit(): void {
        this.editorOption = {
            language: 'yaml',
            minimap: {enabled: false}
        };

        this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectWorkflowEntityComponent.PANEL_KEY)) ?? '50%';

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            this.resizing = resizing;
            this._cd.markForCheck();
        });

        this._cd.markForCheck();
    }

    ngOnChanges(): void {
        this.flatSchema = JSONSchema.flat(this.schema);
        this.dataGraph = this.data;
        this.dataEditor = this.data;
        this._cd.markForCheck();
    }

    ngOnDestroy(): void {
    } // Should be set to use @AutoUnsubscribe with AOT

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({schemas: [{uri: '', schema: this.flatSchema}]});
    }

    onEditorChange(data: string): void {
        this.dataGraph = data;
        this._cd.markForCheck();
    }

    @Debounce(200)
    onFormChange(data: string): void {
        this.dataEditor = data;
        this._cd.markForCheck();
    }

    panelStartResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({resizing: true}));
    }

    panelEndResize(size: number): void {
        this._store.dispatch(new actionPreferences.SavePanelSize({
            panelKey: ProjectWorkflowEntityComponent.PANEL_KEY,
            size: size
        }));
        this._store.dispatch(new actionPreferences.SetPanelResize({resizing: false}));
        this.editor.layout();
    }
}
