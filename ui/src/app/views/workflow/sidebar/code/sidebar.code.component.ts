import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input,
    OnDestroy,
    OnInit,
    ViewChild
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { FlatSchema, JSONSchema } from 'app/model/schema.model';
import { Workflow } from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { UserService } from 'app/service/user/user.service';
import { WorkflowCoreService } from 'app/service/workflow/workflow.core.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchAsCodeWorkflow, GetWorkflow, ImportWorkflow, PreviewWorkflow } from 'app/store/workflow.action';
import * as yaml from 'js-yaml';
import { Schema } from 'js-yaml';
import { Validator } from 'jsonschema';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

declare let CodeMirror: any;

@Component({
    selector: 'app-workflow-sidebar-code',
    templateUrl: './sidebar.code.html',
    styleUrls: ['./sidebar.code.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowSidebarCodeComponent implements OnInit, AfterViewInit, OnDestroy {
    @ViewChild('codeMirror') codemirror: any;

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    // Flag indicate if sidebar is open
    @Input()
    set open(data: boolean) {
        if (data && !this.updated) {
            this.loadingGet = true;
            this.store.dispatch(new FetchAsCodeWorkflow({
                projectKey: this.project.key,
                workflowName: this.workflow.name
            })).pipe(finalize(() => {
                this.loadingGet = false;
                this._cd.markForCheck();
            }))
                .subscribe(() => this.exportedWf = this.workflow.asCode);
        }
        this._open = data;
    }
    get open() {
        return this._open;
    }
    _open = false;


    asCodeEditorSubscription: Subscription;
    codeMirrorConfig: any;
    exportedWf: string;
    updated = false;
    loading = false;
    loadingGet = true;
    previewMode = false;
    themeSubscription: Subscription;
    workflowSchema: Schema;
    flatSchema: FlatSchema;
    viewInit: boolean;

    constructor(
        private store: Store,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _workflowCore: WorkflowCoreService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef,
        private _userService: UserService
    ) {
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            tabSize: 2,
            indentWithTabs: false,
            gutters: ['CodeMirror-lint-markers'],
            lint: {
                getAnnotations: this.workflowCheck
            }
        };

        this._userService.getSchema('workflow').pipe(first()).subscribe(sc => {
            if (sc.workflow) {
                this.workflowSchema = <Schema>JSON.parse(sc.workflow);
                this.flatSchema = JSONSchema.flat(this.workflowSchema);
                if (this.viewInit) {
                    this.initCodeMirror();
                }
            }
        });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    workflowCheck = cm => {
        const errors = CodeMirror.lint.yaml(cm);
        if (errors && errors.length > 0) {
            return errors;
        }
        if (!cm) {
            return [];
        }

        if (!this.workflowSchema) {
            return [];
        }

        const yamlData = yaml.load(cm);
        let v = new Validator();
        let result = v.validate(yamlData, this.workflowSchema);
        return this.toCodemirrorError(<[]>result.errors);
    };

    toCodemirrorError(errors: []) {
        let errs = [];
        if (errors) {
            errors.forEach(e => {
                errs.push({
                    from: {
                        ch: 1,
                        line: 1
                    },
                    message: e['message']
                });
            });
        }
        return errs;
    }

    ngOnInit(): void {
        this.asCodeEditorSubscription = this._workflowCore.getAsCodeEditor()
            .subscribe((state) => {
                if (state != null && state.save) {
                    this.save();
                }
            });

        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    ngAfterViewInit(): void {
        this.viewInit = true;
        if (this.workflowSchema) {
            this.initCodeMirror();
        }
    }

    initCodeMirror(): void {
        this.codemirror.instance.on('keyup', (cm, event) => {
            // 32 : space ; 13: enter ; 8: backspace
            if (event.which > 46 || event.which === 32 || event.which === 13 || event.which === 8) {
                CodeMirror.showHint(cm, CodeMirror.hint.asCode, {
                    completeSingle: true,
                    closeCharacters: / /,
                    specialChars: '',
                    suggests: {
                        pipelines: this.project.pipeline_names.map(n => n.name),
                        applications: this.project.application_names.map(n => n.name),
                        environments: this.project.environment_names.map(n => n.name)
                    },
                    schema: this.flatSchema
                });
            }
        });
    }

    keyEvent(event: KeyboardEvent) {
        if (event.key === 's' && (event.ctrlKey || event.metaKey)) {
            this.save();
            event.preventDefault();
        }
    }

    cancel() {
        if (this.previewMode) {
            this.store.dispatch(new GetWorkflow({
                projectKey: this.project.key,
                workflowName: this.workflow.name
            })).subscribe(() => this._workflowCore.toggleAsCodeEditor({ open: false, save: false }));
            this.previewMode = false;
        } else {
            this._workflowCore.setWorkflowPreview(null);
            this._workflowCore.toggleAsCodeEditor({ open: false, save: false });
        }
        this.updated = false;
    }

    unselectAll() {
        let url = this._router.createUrlTree(['./'], {
            relativeTo: this._activatedRoute,
            queryParams: {}
        });
        this._router.navigateByUrl(url.toString());
    }

    preview() {
        this.unselectAll();
        this.loading = true;
        this.previewMode = true;
        this.store.dispatch(new PreviewWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            wfCode: this.exportedWf
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => this._workflowCore.toggleAsCodeEditor({ open: false, save: false }));
    }

    save() {
        this.unselectAll();
        this.loading = true;
        this.store.dispatch(new ImportWorkflow({
            projectKey: this.project.key,
            wfName: this.workflow.name,
            workflowCode: this.exportedWf
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this.previewMode = false;
                this.updated = false;
                this._workflowCore.toggleAsCodeEditor({ open: false, save: false });
                this._workflowCore.setWorkflowPreview(null);
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }
}
