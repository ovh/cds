import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component, OnInit,
} from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { WorkflowDeletedDependencies } from 'app/model/purge.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { DeleteWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import { finalize } from 'rxjs/operators';
import { NzModalRef } from 'ng-zorro-antd/modal';

@Component({
    selector: 'app-workflow-delete-modal',
    templateUrl: './delete-modal.html',
    styleUrls: ['./delete-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowDeleteModalComponent implements OnInit {
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
        private _workflowService: WorkflowService,
        private _modal: NzModalRef
    ) {}

    ngOnInit() {
        this.project = this.store.selectSnapshot(ProjectState.projectSnapshot);
        this.workflow = this.store.selectSnapshot(WorkflowState.workflowSnapshot);
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
                this._toast.success('', this._translate.instant('workflow_deleted'));
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
                this._modal.destroy();
        });
    }
}
