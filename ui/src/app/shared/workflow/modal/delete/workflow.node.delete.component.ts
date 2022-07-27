import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input,
    OnInit,
    ViewChild
} from '@angular/core';
import { WNode, Workflow } from 'app/model/workflow.model';
import cloneDeep from 'lodash-es/cloneDeep';
import { WorkflowState } from 'app/store/workflow.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { finalize } from 'rxjs/operators';
import { NzModalRef } from 'ng-zorro-antd/modal';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { ToastService } from 'app/shared/toast/ToastService';

@Component({
    selector: 'app-workflow-node-delete',
    templateUrl: './workflow.node.delete.html',
    styleUrls: ['./workflow.node.delete.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowDeleteNodeComponent implements OnInit {

    @Input() project: Project;
    @Input() node: WNode;
    @Input() workflow: Workflow;
    loading: boolean = false;

    deleteAll = 'only';
    isRoot = false;

    constructor(public _modal: NzModalRef, private _store: Store, private _cd: ChangeDetectorRef,
        private _toast: ToastService) { }

    ngOnInit(): void {
        this.isRoot = this.node?.id === this.workflow?.workflow_data?.node?.id;
    }

    deleteNode(): void {
        let clonedWorkflow = cloneDeep(this.workflow);
        clonedWorkflow.notifications = cloneDeep(this.workflow.notifications);
        if (this.deleteAll === 'only') {
            Workflow.removeNodeOnly(clonedWorkflow, this.node.id);
        } else {
            Workflow.removeNodeWithChild(clonedWorkflow, this.node.id);
        }
        this.updateWorkflow(clonedWorkflow);
    }

    updateWorkflow(w: Workflow): void {
        this.loading = true;
        let editMode = this._store.selectSnapshot(WorkflowState).editMode;
        this._store.dispatch(new UpdateWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            changes: w
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(() => {
            if (!editMode) {
                this._toast.success('', 'Workflow updated');
            }
            this._modal.destroy();
        });
    }
}
