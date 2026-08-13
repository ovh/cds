import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, SimpleChanges, ViewChild } from "@angular/core";
import { Store } from "@ngxs/store";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { PreferencesState } from "app/store/preferences.state";
import { editor } from "monaco-editor";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Subscription } from "rxjs";
import { V2WorkflowRun } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { dump } from "js-yaml";
import { NzTreeNode, NzTreeNodeOptions } from "ng-zorro-antd/tree";
import { EntityReferenceKind, EntityReferenceUtils } from "app/shared/entity-reference.utils";

export type SourceFileKind = 'workflow' | 'action' | 'model';

/**
 * CDS v2 entity paths are <projectKey>/<vcsName>/<repoName>/<name>@<ref>, and the
 * ref itself may hold slashes, so the name is read from the part before the '@'.
 */
export class SourceFile {
    /** Raw key as held in the run snapshot, ref included. */
    key: string;
    /** Key without the @ref suffix. */
    path: string;
    /** Entity name, the last segment of the path. */
    name: string;
    /** Project, vcs and repository holding the entity. */
    scope: string;
    /** Ref the entity was resolved at, shortened for display. */
    ref: string;
    kind: SourceFileKind;
    /** Human readable entity type, shown instead of an icon. */
    kindLabel: string;
    content: string;
}

/** A `uses:` or `runs-on:` value that resolves to another file of the snapshot. */
class SourceReference {
    line: number;
    startColumn: number;
    endColumn: number;
    target: number;
}

@Component({
    standalone: false,
    selector: 'app-run-sources',
    templateUrl: './run-sources.html',
    styleUrls: ['./run-sources.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunSourcesComponent implements OnInit, OnChanges, OnDestroy {
    @ViewChild('editor') editor: NzCodeEditorComponent;

    @Input() run: V2WorkflowRun;
    /** Key of the file to open, so a shared link can target a precise source. */
    @Input() selectedFileKey: string;

    @Output() selectedFileKeyChange = new EventEmitter<string>();

    editorOption: EditorOptions;
    resizingSubscription: Subscription;

    files: Array<SourceFile> = [];
    /** Grouped file list rendered as a tree inside the path dropdown. */
    treeNodes: Array<NzTreeNodeOptions> = [];
    /** Visited files, as indexes in `files`. The last entry is the current file. */
    history: Array<number> = [];
    historyIndex: number = -1;

    private _editorInstance: editor.ICodeEditor;
    private _decorations: editor.IEditorDecorationsCollection;
    private _references: Array<SourceReference> = [];

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) { }

    get current(): SourceFile { return this.files[this.currentIndex] ?? null; }
    get currentIndex(): number { return this.history[this.historyIndex] ?? 0; }
    get canBack(): boolean { return this.historyIndex > 0; }
    get canForward(): boolean { return this.historyIndex < this.history.length - 1; }

    get selectedTreeKey(): string { return `${this.currentIndex}`; }

    /** Closed control reads "Worker model · ubuntu-latest @ main"; full path in the tooltip. */
    displayPath = (node: NzTreeNode): string => {
        const file = this.files[parseInt(node.key, 10)];
        if (!file) {
            return node.title;
        }
        return `${file.kindLabel} · ${file.name}` + (file.ref ? ` @ ${file.ref}` : '');
    };

    onTreeSelect(key: string): void {
        if (!key || key.startsWith('group:')) {
            return;
        }
        this.openFile(parseInt(key, 10));
    }

    ngOnChanges(changes: SimpleChanges): void {
        // The parent mirrors back the opened file, so only a new run rebuilds the
        // list. Otherwise every navigation would reset the history.
        if (!changes['run']) {
            if (changes['selectedFileKey'] && this.selectedFileKey && this.selectedFileKey !== this.current?.key) {
                this.openFile(this.files.findIndex(f => f.key === this.selectedFileKey));
            }
            return;
        }

        const previousKey = this.current?.key;

        const workflowKey = `${this.run.vcs_server}/${this.run.repository}/${this.run.workflow_name}`;
        this.files = [RunSourcesComponent.buildFile(workflowKey, 'workflow',
            dump(this.run.workflow_data.workflow, { lineWidth: -1 }))];
        Object.keys(this.run.workflow_data.actions ?? {}).sort().forEach(k => {
            this.files.push(RunSourcesComponent.buildFile(k, 'action',
                dump(this.run.workflow_data.actions[k], { lineWidth: -1 })));
        });
        Object.keys(this.run.workflow_data.worker_models ?? {}).sort().forEach(k => {
            this.files.push(RunSourcesComponent.buildFile(k, 'model',
                dump(this.run.workflow_data.worker_models[k], { lineWidth: -1 })));
        });

        const groups: Array<{ kind: SourceFileKind, title: string }> = [
            { kind: 'workflow', title: 'Workflow' },
            { kind: 'action', title: 'Actions' },
            { kind: 'model', title: 'Worker models' }
        ];
        this.treeNodes = groups.map(g => (<NzTreeNodeOptions>{
            title: g.title,
            key: 'group:' + g.kind,
            expanded: true,
            selectable: false,
            isGroup: true,
            children: this.files
                .map((f, index) => ({ f, index }))
                .filter(entry => entry.f.kind === g.kind)
                .map(entry => (<NzTreeNodeOptions>{
                    title: entry.f.name,
                    key: `${entry.index}`,
                    isLeaf: true,
                    name: entry.f.name,
                    scope: entry.f.scope,
                    ref: entry.f.ref
                }))
        })).filter(node => node.children.length > 0);

        // Honour a shared link, else keep the reader where they were on refresh.
        const wanted = this.selectedFileKey ?? previousKey;
        const restored = wanted ? this.files.findIndex(f => f.key === wanted) : -1;
        this.history = [restored !== -1 ? restored : 0];
        this.historyIndex = 0;

        this.refreshReferences();
        this.selectedFileKeyChange.emit(this.current?.key);
        this._cd.markForCheck();
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.editorOption = {
            language: 'yaml',
            minimap: { enabled: false },
            readOnly: true,
            scrollBeyondLastLine: false,
            ariaLabel: 'Workflow sources viewer'
        };

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            if (!resizing && this.editor) {
                this.editor.layout();
            }
        });
    }

    static readonly KIND_LABELS: { [kind: string]: string } = {
        workflow: 'Workflow',
        action: 'Action',
        model: 'Worker model'
    };

    static buildFile(key: string, kind: SourceFileKind, content: string): SourceFile {
        const at = key.indexOf('@');
        const path = at === -1 ? key : key.substring(0, at);
        const ref = at === -1 ? '' : key.substring(at + 1);
        const separator = path.lastIndexOf('/');
        return <SourceFile>{
            key,
            path,
            name: separator === -1 ? path : path.substring(separator + 1),
            scope: separator === -1 ? '' : path.substring(0, separator),
            ref: ref.replace(/^refs\/(heads|tags)\//, ''),
            kind,
            kindLabel: RunSourcesComponent.KIND_LABELS[kind],
            content
        };
    }

    /** Open a file, dropping any forward history. */
    openFile(index: number): void {
        if (index < 0 || index >= this.files.length || index === this.currentIndex) {
            return;
        }
        this.history = this.history.slice(0, this.historyIndex + 1);
        this.history.push(index);
        this.historyIndex = this.history.length - 1;
        this.navigated();
    }

    back(): void {
        if (this.canBack) {
            this.historyIndex--;
            this.navigated();
        }
    }

    forward(): void {
        if (this.canForward) {
            this.historyIndex++;
            this.navigated();
        }
    }

    private navigated(): void {
        this.refreshReferences();
        this.selectedFileKeyChange.emit(this.current?.key);
        this._cd.markForCheck();
    }

    onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
        this._editorInstance = <editor.ICodeEditor>e;
        // A rebuilt editor needs its own collection; the previous one belongs to a
        // discarded editor and setting on it renders nothing.
        this._decorations = this._editorInstance.createDecorationsCollection();
        // Monaco drops every decoration when the model value is replaced, which
        // happens after navigation pushes the new content through ngModel.
        this._editorInstance.onDidChangeModelContent(() => this.applyDecorations());
        this._editorInstance.onMouseDown(event => {
            const position = event.target?.position;
            if (!position) {
                return;
            }
            const reference = this._references.find(r => r.line === position.lineNumber
                && position.column >= r.startColumn && position.column <= r.endColumn);
            if (reference) {
                this.openFile(reference.target);
            }
        });
        this.editor.layout();
        this.applyDecorations();
    }

    /**
     * Locate every `uses:` / `runs-on:` value of the current file that resolves to
     * another file of the snapshot, so it can be rendered and clicked as a link.
     */
    private refreshReferences(): void {
        this._references = [];
        EntityReferenceUtils.scan(this.current?.content).forEach(reference => {
            const target = this.resolve(reference.kind, reference.value);
            if (target !== -1) {
                this._references.push(<SourceReference>{
                    line: reference.line,
                    startColumn: reference.startColumn,
                    endColumn: reference.endColumn,
                    target
                });
            }
        });
        this.applyDecorations();
    }

    /** Snapshot keys are full paths; references may use a shorter form and an @ref suffix. */
    private resolve(kind: EntityReferenceKind, reference: string): number {
        const wanted = reference.split('@')[0].trim();
        if (!wanted) {
            return -1;
        }
        const wantedName = wanted.substring(wanted.lastIndexOf('/') + 1);
        const candidates = this.files.filter(f => f.kind === kind);
        const found = candidates.find(f => f.path === wanted)
            ?? candidates.find(f => f.path.endsWith('/' + wanted))
            ?? candidates.find(f => f.name === wantedName);
        return found ? this.files.indexOf(found) : -1;
    }

    private applyDecorations(): void {
        if (!this._decorations) {
            return;
        }
        this._decorations.set(this._references.map(r => ({
            range: { startLineNumber: r.line, startColumn: r.startColumn, endLineNumber: r.line, endColumn: r.endColumn },
            options: { inlineClassName: 'cds-source-link', hoverMessage: { value: 'Open definition' } }
        })));
    }
}
