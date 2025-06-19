import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from "@angular/core";
import { Router } from "@angular/router";
import { Store } from "@ngxs/store";
import { editor } from 'monaco-editor';
import { Schema } from 'app/model/json-schema.model';
import { Concurrency, ProjectConcurrencyRuns } from "app/model/project.concurrency.model";
import { Project, ProjectRunRetention } from "app/model/project.model";
import { FlatSchema, JSONSchema } from "app/model/schema.model";
import { V2ProjectService } from "app/service/projectv2/project.service";
import { ErrorUtils } from "app/shared/error.utils";
import * as actionPreferences from 'app/store/preferences.action';
import { NzMessageService } from "ng-zorro-antd/message";
import { finalize, first, forkJoin, lastValueFrom, pipe } from "rxjs";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { dump, load, LoadOptions, YAMLException } from "js-yaml";

declare const monaco: any;

@Component({
    selector: 'app-project-run-retention',
    templateUrl: './retention.html',
    styleUrls: ['./retention.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectRunRetentionComponent implements OnInit {
    static PANEL_KEY = 'project-v2-run-retention';

    @Input() project: Project;
    @ViewChild('editor') editor: NzCodeEditorComponent;

    jsonSchema: Schema;
    jsonFlatSchema: FlatSchema
    retention: ProjectRunRetention;
    dataEditor: string;
    editorOption: EditorOptions;

    constructor(private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _store: Store,
        private _v2ProjectService: V2ProjectService) {

    }

    ngOnInit(): void {
        this.editorOption = {
            language: 'yaml',
            minimap: { enabled: false }
        };
        this._v2ProjectService.getRetentionSchema(this.project.key).pipe(first()).subscribe(j => {
            this.jsonSchema = j;
            this._cd.markForCheck();
        });
        this._v2ProjectService.getRetention(this.project.key).pipe(first()).subscribe(r => {
            this.retention = r;
        })
        forkJoin([
            this._v2ProjectService.getRetention(this.project.key),
            this._v2ProjectService.getRetentionSchema(this.project.key)
        ]).pipe(finalize(() => {
            this._cd.markForCheck();
        })).subscribe(result => {
            this.retention = result[0];
            this.dataEditor = dump(this.retention.retentions);
            this.jsonFlatSchema = JSONSchema.flat(result[1]);
            
        });
    }

    onEditorChange(event: string) {
        console.log(event);
        this.dataEditor = event;
        this._cd.markForCheck();
    }

    updateRetention(): void {
        try {
            this.retention.retentions = load(this.dataEditor && this.dataEditor !== '' ? this.dataEditor : '{}', <LoadOptions>{
                onWarning: (e) => {}
            });
            this._v2ProjectService.updateRetention(this.project.key, this.retention).pipe(first()).subscribe(() => {
                this._messageService.success('Project run retention updated');
            })
        } catch (e) {
            this._messageService.error('Invalid yaml data');
        }
    }

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({ schemas: [{ uri: '', schema: this.jsonFlatSchema }] });
        this.editor.layout();
    }
}