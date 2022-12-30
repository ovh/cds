import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy} from "@angular/core";
import {NzConfigService} from "ng-zorro-antd/core/config";
import {CodeEditorConfig} from "ng-zorro-antd/core/config/config";
import {ThemeStore} from "../../service/theme/theme.store";
import {AutoUnsubscribe} from "../decorator/autoUnsubscribe";
import {Subscription} from "rxjs/Subscription";
import {
    editor,
    Uri
} from 'monaco-editor';
import {JSONSchema} from "../../model/schema.model";
import {Schema} from "jsonschema";
import {Editor} from "../../model/editor.model";

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

    _jsonSchema: Schema;

    @Input() set jsonschema(schema: Schema) {
        this._jsonSchema = schema;
    }

    get jsonschema(): Schema {
        return this._jsonSchema;
    }

    themeSub: Subscription;
    editor?: editor.ICodeEditor | editor.IEditor;

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
    }

    ngOnDestroy() {}

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        console.log('On editor init');
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({schemas: [{uri: '', schema: JSONSchema.flat(this._jsonSchema)}]});
        monaco.languages.registerCompletionItemProvider("yaml", Editor.completionProvider(monaco));
    }

}
