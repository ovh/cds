import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { PermissionValue } from 'app/model/permission.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkflowCoreService } from 'app/service/workflow/workflow.core.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchAsCodeWorkflow, GetWorkflow, ImportWorkflow, PreviewWorkflow } from 'app/store/workflow.action';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-sidebar-code',
    templateUrl: './sidebar.code.html',
    styleUrls: ['./sidebar.code.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarCodeComponent implements OnInit {
    @ViewChild('codeMirror', {static: false}) codemirror: any;

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    // Flag indicate if sidebar is open
    @Input('open')
    set open(data: boolean) {
        if (data && !this.updated) {
            this.loadingGet = true;
            this.store.dispatch(new FetchAsCodeWorkflow({
                projectKey: this.project.key,
                workflowName: this.workflow.name
            })).pipe(finalize(() => this.loadingGet = false))
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
    permissionEnum = PermissionValue;
    themeSubscription: Subscription;

    constructor(
        private store: Store,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _workflowCore: WorkflowCoreService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _theme: ThemeStore
    ) {
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };
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
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => this._workflowCore.toggleAsCodeEditor({ open: false, save: false }));
    }

    save() {
        this.unselectAll();
        this.loading = true;
        this.store.dispatch(new ImportWorkflow({
            projectKey: this.project.key,
            wfName: this.workflow.name,
            workflowCode: this.exportedWf
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.previewMode = false;
                this.updated = false;
                this._workflowCore.toggleAsCodeEditor({ open: false, save: false });
                this._workflowCore.setWorkflowPreview(null);
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }
}
