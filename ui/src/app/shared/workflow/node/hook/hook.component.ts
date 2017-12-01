import {AfterViewInit, Component, ElementRef, Input, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeHook} from '../../../../model/workflow.model';
import {WorkflowService} from '../../../../service/workflow/workflow.service';
import {ToastService} from '../../../toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {WorkflowNodeHookFormComponent} from './form/node.hook.component';
import {Project} from '../../../../model/project.model';
import {HookEvent} from './hook.event';
import {cloneDeep} from 'lodash';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
})
export class WorkflowNodeHookComponent implements AfterViewInit {

    @Input() hook: WorkflowNodeHook;
    @Input() readonly = false;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() node: WorkflowNode;

    @ViewChild('editHook')
    editHook: WorkflowNodeHookFormComponent;

    loading = false;

    constructor(private elementRef: ElementRef, private _workflowStore: WorkflowStore, private _toast: ToastService,
                private _translate: TranslateService) {
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = '5px';
    }

    openEditHookModal(): void {
        if (this.editHook) {
            this.editHook.show();
        }
    }

    updateHook(h: HookEvent): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        this.loading = true;
        if (h.type === 'delete') {
            Workflow.removeHook(workflowToUpdate, h.hook);
        } else {
            Workflow.updateHook(workflowToUpdate, h.hook);

        }
        this._workflowStore.updateWorkflow(workflowToUpdate.project_key, workflowToUpdate).pipe(finalize(() => {
            this.loading = false;
        })).subscribe(() => {
            this.editHook.modal.approve(true);
            this._toast.success('', this._translate.instant('workflow_updated'));
        });
    }
}
