import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { NzConfigService } from "ng-zorro-antd/core/config";
import { CodeEditorConfig } from "ng-zorro-antd/core/config/config";
import { ThemeStore } from "../../service/theme/theme.store";
import { AutoUnsubscribe } from "../decorator/autoUnsubscribe";
import { Subscription } from "rxjs/Subscription";
import {
    editor,
} from 'monaco-editor';
import { FlatSchema, JSONSchema } from "../../model/schema.model";
import { Schema } from "jsonschema";
import { EditorOptions } from "ng-zorro-antd/code-editor/typings";
import { NzCodeEditorComponent } from "ng-zorro-antd/code-editor";

declare const monaco: any;

@Component({
    selector: 'app-entity',
    templateUrl: './entity.form.component.html',
    styleUrls: ['./entity.form.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class EntityFormComponent implements OnInit, OnChanges, OnDestroy {
    @ViewChild('editor') editor: NzCodeEditorComponent;

    @Input() path: string;
    @Input() data: string;
    @Input() schema: Schema;

    flatSchema: FlatSchema;
    themeSub: Subscription;
    editorOption: EditorOptions;
    resizing: boolean;

    constructor(
        private _cd: ChangeDetectorRef,
        private _configService: NzConfigService,
        private _themeStore: ThemeStore
    ) { }

    ngOnInit(): void {
        this.themeSub = this._themeStore.get().subscribe(t => {
            const config: CodeEditorConfig = this._configService.getConfigForComponent('codeEditor') || {};
            this._configService.set('codeEditor', {
                defaultEditorOption: {
                    ...config,
                    theme: t === 'light' ? 'vs' : 'vs-dark',
                },
            });
        });
        this.editorOption = {
            language: 'yaml',
            minimap: { enabled: false }
        };
        this._cd.markForCheck();
    }

    ngOnChanges(): void {
        this.flatSchema = JSONSchema.flat(this.schema);
        this._cd.markForCheck();
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({ schemas: [{ uri: '', schema: this.flatSchema }] });
    }

    panelStartResize(): void {
        this.resizing = true;
        this._cd.markForCheck();
    }

    panelEndResize(): void {
        this.resizing = false;
        this._cd.markForCheck();
        setTimeout(() => { window.dispatchEvent(new Event('resize')) });
    }
}
