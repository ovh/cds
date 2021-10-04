import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ViewChild
} from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Project } from 'app/model/project.model';
import { WorkflowDeletedDependencies } from 'app/model/purge.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { DeleteWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import { finalize } from 'rxjs/operators';

class WorkflowDeleteModalComponentDependency {
    type: string; // application, pipeline, environment
    name: string;
}

@Component({
    selector: 'app-workflow-delete-modal',
    templateUrl: './delete-modal.html',
    styleUrls: ['./delete-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowDeleteModalComponent {
    @ViewChild('workflowDeleteModal') workflowDeleteModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    open: boolean;
    loading: boolean;
    dependencies: WorkflowDeletedDependencies;
    project: Project;
    workflow: Workflow;

    constructor(
        private store: Store,
        private _cd: ChangeDetectorRef,
        public _translate: TranslateService,
        private _toast: ToastService,
        private _router: Router,
        private _modalService: SuiModalService,
        private _workflowService: WorkflowService
    ) {}

    show() {
        if (this.open) {
            return;
        }

        this.open = true;
        this.project = this.store.selectSnapshot(ProjectState.projectSnapshot);
        this.workflow = this.store.selectSnapshot(WorkflowState.workflowSnapshot);
        const config = new TemplateModalConfig<boolean, boolean, void>(this.workflowDeleteModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(_ => {
             this.closeCallback();
        });
        this.modal.onDeny(_ => {
            this.closeCallback();
        });
        this._workflowService.getDeletedDependencies(this.workflow).pipe(
            finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((x) => {
                this.dependencies = x;
            }
        );
    }

    deleteWorkflow(b: boolean): void {
        this.loading = true;
        this.store.dispatch(
            new DeleteWorkflow({
                    projectKey: this.project.key,
                    workflowName: this.workflow.name,
                    withDependencies: b
                })
        ).pipe(
            finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this.modal.approve(true);
                this._toast.success('', this._translate.instant('workflow_deleted'));
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
        });
    }

    closeCallback(): void {
        this.open = false;
    }
}
