import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy} from "@angular/core";
import {NzConfigService} from "ng-zorro-antd/core/config";
import {CodeEditorConfig} from "ng-zorro-antd/core/config/config";
import {ThemeStore} from "../../service/theme/theme.store";
import {AutoUnsubscribe} from "../decorator/autoUnsubscribe";
import {Subscription} from "rxjs/Subscription";
import {
    editor,
} from 'monaco-editor';
import {FlatSchema, JSONSchema} from "../../model/schema.model";
import {Schema} from "jsonschema";
import {JoinedEditorOptions} from "ng-zorro-antd/code-editor/typings";

declare const monaco: any;

@Component({
    selector: 'app-entity',
    templateUrl: './entity.form.component.html',
    styleUrls: ['./entity.form.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class EntityFormComponent implements OnDestroy {

    @Input() data: string;

    _jsonSchema: FlatSchema;
    @Input() set jsonschema(schema: Schema) {
        this._jsonSchema = JSONSchema.flat(schema);
    }

    themeSub: Subscription;
    editorOption: JoinedEditorOptions;

    constructor(private _cd: ChangeDetectorRef, private _configService: NzConfigService, private _themeStore: ThemeStore) {
        this.themeSub = this._themeStore.get().subscribe(t => {
            const config: CodeEditorConfig = this._configService.getConfigForComponent('codeEditor') || {};
            this._configService.set('codeEditor', {
                defaultEditorOption: {
                    ...config,
                    theme: t === 'light' ? 'vs' : 'vs-dark'
                },
            });
        });
        this.editorOption = {
            language: 'yaml',
            minimap: {enabled: false},
        }
    }

    ngOnDestroy() {}

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({schemas: [{uri: '', schema: this._jsonSchema}]});
    }

}
