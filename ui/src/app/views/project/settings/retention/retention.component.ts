import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { editor } from 'monaco-editor';
import { Schema } from 'app/model/json-schema.model';
import { Project, ProjectRunRetention } from "app/model/project.model";
import { FlatSchema, JSONSchema } from "app/model/schema.model";
import { V2ProjectService } from "app/service/projectv2/project.service";
import { NzMessageService } from "ng-zorro-antd/message";
import { finalize, first, forkJoin, Subscription } from "rxjs";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { dump, load, LoadOptions } from "js-yaml";
import { EventV2Service } from "app/event-v2.service";
import { WebsocketV2Filter, WebsocketV2FilterType } from "app/model/websocket-v2";
import { EventV2State } from "app/store/event-v2.state";
import { EventV2Type } from "app/model/event-v2.model";

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

    dryRunVisible: boolean = false;
    dryRunReport: any;
    dryRunReportID: string;

    reportID: string;
    reportLoading: boolean = false;

    eventV2Subscription: Subscription;

    constructor(private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _eventV2Service: EventV2Service,
        private _store: Store,
        private _v2ProjectService: V2ProjectService) {

        this.eventV2Subscription = this._store.select(EventV2State.last).subscribe((event) => {
            if (event && event?.type == EventV2Type.EventProjectPurge) {
                if (this.dryRunReportID && event?.payload?.id === this.dryRunReportID) {
                    this.dryRunReport = event.payload;
                    this._cd.markForCheck();
                } else if (this.reportID && event?.payload?.id === this.reportID) {
                    // reload retention
                    this._v2ProjectService.getRetention(this.project.key).pipe(first(), finalize(() => {
                        this._cd.markForCheck();
                    })).subscribe(r => {
                        this.retention = r;
                        this.reportLoading = false;
                    })
                    this._cd.markForCheck();
                }
            }
           
        });

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
        this.dataEditor = event;
        this._cd.markForCheck();
    }

    runDryRunRetention(): void {
         try {
            this.retention.retentions = load(this.dataEditor && this.dataEditor !== '' ? this.dataEditor : '{}', <LoadOptions>{
                onWarning: (e) => {}
            });
            this._v2ProjectService.updateRetention(this.project.key, this.retention).pipe(first()).subscribe(() => {
                this._messageService.success('Project run retention updated');
            });
        } catch (e) {
            this._messageService.error('Invalid yaml data');
            return;
        }
        this._v2ProjectService.runDryRunRetention(this.project.key, this.retention).pipe(first(), finalize(() => {
            this._cd.markForCheck();
        })).subscribe(e => {
            this._messageService.success('Dry run started, please wait');
            this._eventV2Service.updateFilter(<WebsocketV2Filter>{
                type: WebsocketV2FilterType.PROJECT_PURGE_REPORT,
                project_key: this.project.key,
                purge_report_id: e.report_id
            });
            this.dryRunReportID = e.report_id;
            this.dryRunVisible = true;
        });
    }

    runRetention(): void {
        this._v2ProjectService.runRetention(this.project.key, this.retention).pipe(first(), finalize(() => {
            this._cd.markForCheck();
        })).subscribe(e => {
            this._messageService.success('Run retention started, please wait');
            this.reportID = e.report_id;
            this.reportLoading = true;
            this._eventV2Service.updateFilter(<WebsocketV2Filter>{
                type: WebsocketV2FilterType.PROJECT_PURGE_REPORT,
                project_key: this.project.key,
                purge_report_id: e.report_id
            });
        });
    }

    updateRetention(): void {
        try {
            this.retention.retentions = load(this.dataEditor && this.dataEditor !== '' ? this.dataEditor : '{}', <LoadOptions>{
                onWarning: (e) => {}
            });
            this._v2ProjectService.updateRetention(this.project.key, this.retention).pipe(first()).subscribe(() => {
                this._messageService.success('Project run retention updated');
            });
        } catch (e) {
            this._messageService.error('Invalid yaml data');
        }
    }

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({ schemas: [{ uri: '', schema: this.jsonFlatSchema }] });
        this.editor.layout();
    }

    closeModal(): void {
        this.dryRunVisible = false;
        this._cd.markForCheck();
    }
}