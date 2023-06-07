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
import {PanelDirection} from "../../../../../../../shared/resizable-panel/resizable-panel.component";
import {dump, load, LoadOptions} from "js-yaml";
import {EntityAction, EntityWorkflow} from "../../../../../../../model/entity.model";
import {EntityService} from "../../../../../../../service/entity/entity.service";
import {first} from "rxjs/operators";

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
    @Input() workflowSchema: Schema;
    @Input() jobSchema: Schema;
    @Input() disabled: boolean;
    @Input() parentType: string;

    workflowFlatSchema: FlatSchema;
    jobFlatSchema: FlatSchema
    dataGraph: string;
    dataEditor: string;
    jobForm: string;
    editorOption: EditorOptions;
    panelSize: number | string;
    resizing: boolean;
    resizingSubscription: Subscription;
    selectedJob: string;
    actionEntity = EntityAction;

    syntaxErrors: string[];

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _entityService: EntityService
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
        this.workflowFlatSchema = JSONSchema.flat(this.workflowSchema);
        this.jobFlatSchema = JSONSchema.flat(this.jobSchema);
        this.dataGraph = this.data;
        this.dataEditor = this.data;
        this._cd.markForCheck();
    }

    ngOnDestroy(): void {
    } // Should be set to use @AutoUnsubscribe with AOT

    selectJob(jobName: string) {
        this.selectedJob = jobName;
        this.updateSelectedJob();
        this._cd.markForCheck();
    }

    updateSelectedJob(): void {
        let workflowForm: any;
        try {
            workflowForm = load(this.dataGraph && this.dataGraph !== '' ? this.dataGraph : '{}', <LoadOptions>{
                onWarning: (e) => {
                }
            });
        } catch (e) {
            console.error("Invalid workflow:", this.dataGraph)
        }
        if (workflowForm && workflowForm['jobs'] && workflowForm['jobs'][this.selectedJob]) {
            this.jobForm = dump(workflowForm['jobs'][this.selectedJob]);
        }
    }

    @Debounce(500)
    onEditorChange(data: string): void {
        // Call api to check syntax
        this._entityService.checkEntity(EntityWorkflow, data).pipe(first()).subscribe(resp => {
            if (resp?.messages?.length > 0) {
                this.syntaxErrors = resp.messages;
            } else {
                delete this.syntaxErrors
                this.dataGraph = data;
                if (this.selectedJob) {
                    this.updateSelectedJob();
                }
            }
            this._cd.markForCheck();
        })

    }

    @Debounce(200)
    onFormChange(data: string): void {
        let jobYml: any;
        try {
            jobYml = load(data && data !== '' ? data : '{}', <LoadOptions>{
                onWarning: (e) => {
                }
            });
        } catch (e) {
            console.error("Invalid job:", data);
            return;
        }

        let workflowYml: any;
        try {
            workflowYml = load(this.dataGraph && this.dataGraph !== '' ? this.dataGraph : '{}', <LoadOptions>{
                onWarning: (e) => {
                }
            });
        } catch (e) {
            console.error("Invalid workflow:", this.dataGraph);
            return;
        }
        if (workflowYml && workflowYml['jobs'] && workflowYml['jobs'][this.selectedJob]) {
            workflowYml['jobs'][this.selectedJob] = jobYml
        }
        this.dataGraph = dump(workflowYml);
        this.dataEditor = this.dataGraph;

        this._cd.markForCheck();
    }

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({schemas: [{uri: '', schema: this.workflowFlatSchema}]});
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
