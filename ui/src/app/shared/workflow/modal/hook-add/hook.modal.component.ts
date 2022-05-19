import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowNodeHookFormComponent } from 'app/shared/workflow/wizard/hook/hook.form.component';
import { WorkflowState } from 'app/store/workflow.state';
import { NzModalRef } from 'ng-zorro-antd/modal';
import { AddHookWorkflow } from 'app/store/workflow.action';
import { finalize } from 'rxjs/operators';
import { ToastService } from 'app/shared/toast/ToastService';
@Component({
    selector: 'app-hook-modal',
    templateUrl: './hook.modal.html',
    styleUrls: ['./hook.modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowHookModalComponent implements OnDestroy {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;

    loading: boolean;
    editMode: boolean;

    @ViewChild('hookFormComponent')
    hookFormComponent: WorkflowNodeHookFormComponent;

    constructor(private _modal: NzModalRef, private _store: Store, private _cd: ChangeDetectorRef, private _toast: ToastService) {
        this.editMode = this._store.selectSnapshot(WorkflowState).editMode;
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    saveHook(): void {
        this.loading = true;
        let action = new AddHookWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            hook: this.hookFormComponent.hook
        });
        this._store.dispatch(action).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(() => {
            if (!this.editMode) {
                this._toast.success('', 'Workflow updated');
            } else {
                this._toast.info('', 'Draft updated');
            }
            this._modal.triggerOk().then();
        });
    }

    close(): void {
        this._modal.destroy(true)
    }
}
