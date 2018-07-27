import {Component, Input, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/operators';
import {PermissionValue} from '../../../../model/permission.model';
import {Project} from '../../../../model/project.model';
import {Workflow} from '../../../../model/workflow.model';
import {WorkflowCoreService} from '../../../../service/workflow/workflow.core.service';
import {WorkflowService} from '../../../../service/workflow/workflow.service';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-sidebar-code',
    templateUrl: './sidebar.code.html',
    styleUrls: ['./sidebar.code.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarCodeComponent {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    // Flag indicate if sidebar is open
    @Input('open')
    set open(data: boolean) {
        if (data && !this.updated) {
            this.loadingGet = true;
            this._workflowService.getWorkflowExport(this.project.key, this.workflow.name)
                .pipe(finalize(() => this.loadingGet = false))
                .subscribe((wf) => this.exportedWf = wf);
        }
        this._open = data;
    }
    get open() {
        return this._open;
    }
    _open = false;

    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    asCodeEditorSubscription: Subscription;
    codeMirrorConfig: any;

    exportedWf: string;
    updated = false;
    loading = false;
    loadingGet = true;
    permissionEnum = PermissionValue;

    constructor(
        private _workflowCore: WorkflowCoreService,
        private _workflowService: WorkflowService,
        private _workflowStore: WorkflowStore,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };

        this.asCodeEditorSubscription = this._workflowCore.getAsCodeEditor()
            .subscribe((state) => {
                if (state != null && state.save) {
                    this.save();
                }
            });
    }

    cancel() {
        this._workflowCore.setWorkflowPreview(null);
        this._workflowCore.toggleAsCodeEditor({open: false, save: false});
    }

    preview() {
        this.loading = true;
        this._workflowService.previewWorkflowImport(this.project.key, this.exportedWf)
            .pipe(finalize(() => this.loading = false))
            .subscribe((wf) => this._workflowCore.setWorkflowPreview(wf));
    }

    save() {
        this.loading = true;
        this._workflowStore.importWorkflow(this.project.key, this.workflow.name, this.exportedWf)
            .pipe(finalize(() => this.loading = false))
            .subscribe((wf) => {
                this._workflowCore.toggleAsCodeEditor({open: false, save: false});
                this._workflowCore.setWorkflowPreview(null);
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }
}
