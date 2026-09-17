import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnChanges, OnDestroy, SimpleChanges, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { NzDrawerService } from 'ng-zorro-antd/drawer';
import { EditorOptions, NzCodeEditorComponent } from 'ng-zorro-antd/code-editor';
import { editor } from 'monaco-editor';
import { lastValueFrom } from 'rxjs';
import { Store } from '@ngxs/store';
import { load as loadYaml } from 'js-yaml';
import { PreferencesState } from 'app/store/preferences.state';
import * as actionPreferences from 'app/store/preferences.action';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Entity, EntityType, EntityTypeUtil } from 'app/model/entity.model';
import { Schema } from 'app/model/json-schema.model';
import { JSONSchema } from 'app/model/schema.model';
import { ProjectService } from 'app/service/project/project.service';
import { EntityReference, EntityReferenceUtils } from 'app/shared/entity-reference.utils';
import { RepositoryContextService } from './repository-context.service';
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from '../../run-start/run-start.component';

declare const monaco: any;

/**
 * The definition of one entity, read-only, with its json schema and the `uses:` / `runs-on:`
 * values offered as links. The editor stays mounted from one entity to the next: building monaco
 * again for each of them is what made opening one slow.
 */
@Component({
    standalone: false,
    selector: 'app-projectv2-repository-entity-detail',
    templateUrl: './repository-entity-detail.html',
    styleUrls: ['./repository-entity-detail.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryEntityDetailComponent implements OnChanges, OnDestroy {
    /** The entity to show, as the list knows it: its content is read here. */
    @Input() entity: Entity;
    @Input() ref: string;

    @ViewChild('editor') editor: NzCodeEditorComponent;

    ctx = inject(RepositoryContextService);
    content: Entity;
    loading: boolean;
    error: string;
    isWorkflow: boolean;
    /** A workflow written out in full can be drawn; one built from a template cannot. */
    canPreview: boolean;
    mode: 'yaml' | 'preview';
    readonly modes = [{ label: 'YAML', value: 'yaml' }, { label: 'Preview', value: 'preview' }];
    editorOption: EditorOptions = {
        language: 'yaml',
        minimap: { enabled: false },
        readOnly: true,
        scrollBeyondLastLine: false,
        ariaLabel: 'Entity definition editor'
    };

    private _jsonSchema: Schema;
    private _editorInstance: editor.ICodeEditor;
    private _decorations: editor.IEditorDecorationsCollection;
    private _references: Array<EntityReference> = [];

    private _cd = inject(ChangeDetectorRef);
    private _projectService = inject(ProjectService);
    private _router = inject(Router);
    private _drawerService = inject(NzDrawerService);
    private _store = inject(Store);

    constructor() {
        this.mode = this._store.selectSnapshot(PreferencesState.entityDetailMode);
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnChanges(changes: SimpleChanges): void {
        if (!this.entity || !this.ref) {
            return;
        }
        const previous = changes['entity']?.previousValue as Entity;
        const sameEntity = previous && previous.type === this.entity.type && previous.name === this.entity.name && !changes['ref'];
        if (!sameEntity) {
            this.load();
        }
    }

    async load() {
        const { type, name } = this.entity;
        const ref = this.ref;
        this.loading = true;
        this.error = null;
        this._cd.markForCheck();

        try {
            const [jsonSchema, content] = await Promise.all([
                lastValueFrom(this._projectService.getJSONSchema(type)),
                lastValueFrom(this._projectService.getRepoEntity(this.ctx.project.key, this.ctx.vcsName, this.ctx.repoName, type, name, ref))
            ]);
            // Another entity was picked while this one was being read
            if (type !== this.entity?.type || name !== this.entity?.name || ref !== this.ref) {
                return;
            }
            this._jsonSchema = jsonSchema;
            this.content = content;
            this.isWorkflow = content.type === EntityType.Workflow;
            this.canPreview = this.isWorkflow && !loadYaml(content.data)?.['from'];
            // The editor outlives the entity it shows, so the schema of its type is applied on every
            // load and not only when the editor is built.
            this.applyJsonSchema();
        } catch (e: any) {
            this.content = null;
            this.error = e instanceof HttpErrorResponse ? e.error?.message ?? e.message : String(e);
        }

        this.loading = false;
        this._cd.markForCheck();
    }

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        this.applyJsonSchema();
        this._editorInstance = <editor.ICodeEditor>e;
        // The editor is destroyed and rebuilt on every entity load, so the collection
        // has to be rebound; the previous one belongs to a discarded editor.
        this._decorations = this._editorInstance.createDecorationsCollection();
        // Monaco drops decorations whenever the model value is replaced.
        this._editorInstance.onDidChangeModelContent(() => this.applyDecorations());
        this._editorInstance.onMouseDown(event => {
            const position = event.target?.position;
            if (!position) {
                return;
            }
            const reference = this._references.find(r => r.line === position.lineNumber
                && position.column >= r.startColumn && position.column <= r.endColumn);
            if (reference) {
                this.openReference(reference);
            }
        });
        this.editor.layout();
        this.applyDecorations();
    }

    /** Monaco does not follow its container: called when the panels around it are resized. */
    layout(): void {
        this.editor?.layout();
    }

    /** Yaml or graph; the choice is kept for the next workflows. The editor stays mounted, hidden. */
    changeMode(mode: 'yaml' | 'preview'): void {
        this.mode = mode;
        this._store.dispatch(new actionPreferences.SaveEntityDetailMode({ mode }));
        this._cd.markForCheck();
        if (mode === 'yaml') {
            // The editor measures itself once its box is visible again
            setTimeout(() => this.layout());
        }
    }

    /** What the panel shows: the graph only for a workflow that can be drawn and asks for it. */
    get showPreview(): boolean {
        return this.mode === 'preview' && this.canPreview;
    }

    /** Follow a `uses:` / `runs-on:` value to the entity it denotes. */
    openReference(reference: EntityReference): void {
        const path = EntityReferenceUtils.parse(reference.value);
        if (!path) {
            return;
        }
        const entityType = reference.kind === 'model' ? EntityType.WorkerModel : EntityType.Action;
        const ref = path.ref ?? this.ref;
        this._router.navigate([
            '/project', path.projectKey ?? this.ctx.project.key,
            'explore',
            'vcs', path.vcs ?? this.ctx.vcsName,
            'repository', path.repository ?? this.ctx.repoName,
            EntityTypeUtil.toURLParam(entityType), path.name
        ], ref ? { queryParams: { ref } } : {});
    }

    get workflowPath(): string {
        return `${this.ctx.repositoryPath}/${this.entity.name}`;
    }

    openRunStartDrawer(): void {
        this._drawerService.create<ProjectV2RunStartComponent, { params: ProjectV2RunStartComponentParams }, string>({
            nzTitle: 'Start new Workflow Run',
            nzContent: ProjectV2RunStartComponent,
            nzContentParams: {
                params: <ProjectV2RunStartComponentParams>{
                    workflow: this.workflowPath,
                    workflow_ref: this.ref
                }
            },
            nzSize: 'large',
            nzBodyStyle: { 'padding': '0' }
        });
    }

    private applyJsonSchema(): void {
        if (!this._jsonSchema || typeof monaco === 'undefined') {
            return;
        }
        monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
            schemas: [{
                uri: '',
                schema: JSONSchema.flat(this._jsonSchema)
            }]
        });
    }

    private applyDecorations(): void {
        if (!this._decorations) {
            return;
        }
        // Only values that denote a CDS entity are offered as links.
        this._references = EntityReferenceUtils.scan(this.content?.data)
            .filter(r => !!EntityReferenceUtils.parse(r.value));
        this._decorations.set(this._references.map(r => ({
            range: { startLineNumber: r.line, startColumn: r.startColumn, endLineNumber: r.line, endColumn: r.endColumn },
            options: { inlineClassName: 'cds-source-link', hoverMessage: { value: 'Open definition' } }
        })));
    }
}
