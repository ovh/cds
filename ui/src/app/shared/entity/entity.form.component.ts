import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { AutoUnsubscribe } from "../decorator/autoUnsubscribe";
import {
    editor,
} from 'monaco-editor';
import { FlatSchema, JSONSchema } from "../../model/schema.model";
import { EditorOptions } from "ng-zorro-antd/code-editor/typings";
import { NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Store } from "@ngxs/store";
import { PreferencesState } from "app/store/preferences.state";
import * as actionPreferences from 'app/store/preferences.action';
import { Subscription } from 'rxjs';
import Debounce from "../decorator/debounce";
import { Schema } from "app/model/json-schema.model";

declare const monaco: any;

@Component({
    selector: 'app-entity',
    templateUrl: './entity.form.component.html',
    styleUrls: ['./entity.form.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class EntityFormComponent implements OnInit, OnChanges, OnDestroy {
    static PANEL_KEY = 'project-v2-entity-form';

    @ViewChild('editor') editor: NzCodeEditorComponent;

    @Input() path: string;
    @Input() data: string;
    @Input() schema: Schema;
    @Input() disabled: boolean;

    flatSchema: FlatSchema;
    dataForm: string;
    dataEditor: string;
    editorOption: EditorOptions;
    panelSize: number;
    resizing: boolean;
    resizingSubscription: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) { }

    ngOnInit(): void {
        this.editorOption = {
            language: 'yaml',
            minimap: { enabled: false }
        };

        this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(EntityFormComponent.PANEL_KEY)) ?? 600;

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            this.resizing = resizing;
            this._cd.markForCheck();
        });

        this._cd.markForCheck();
    }

    ngOnChanges(): void {
        this.flatSchema = JSONSchema.flat(this.schema);
        this.dataForm = this.data;
        this.dataEditor = this.data;
        this._cd.markForCheck();
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({ schemas: [{ uri: '', schema: this.flatSchema }] });
    }

    onEditorChange(data: string): void {
        this.dataForm = data;
        this._cd.markForCheck();
    }

    @Debounce(200)
    onFormChange(data: string): void {
        this.dataEditor = data;
        this._cd.markForCheck();
    }

    panelStartResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
    }

    panelEndResize(size: number): void {
        this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: EntityFormComponent.PANEL_KEY, size: size }));
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
        this.editor.layout();
    }
}
