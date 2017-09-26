import {AfterViewInit, Component, ElementRef, Input, ViewChild} from '@angular/core';
import {Workflow, WorkflowNodeHook} from '../../../../model/workflow.model';
import {WorkflowService} from '../../../../service/workflow/workflow.service';
import {ToastService} from '../../../toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {WorkflowNodeHookFormComponent} from './form/node.hook.component';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
})
export class WorkflowNodeHookComponent implements AfterViewInit {

    @Input() hook: WorkflowNodeHook;
    @Input() readonly = false;
    @Input() workflow: Workflow;

    @ViewChild('editHook')
    editHook: WorkflowNodeHookFormComponent;

    loading = false;

    constructor(private elementRef: ElementRef, private _workflowService: WorkflowService, private _toast: ToastService,
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

    updateHook(h: WorkflowNodeHook): void {
        this.loading = true;
        Workflow.updateHook(this.workflow, h);
        this._workflowService.updateWorkflow(this.workflow.project_key, this.workflow).finally(() => {
            this.loading = false;
        }).subscribe(() => {
            this.editHook.modal.approve(true);
            this._toast.success('', this._translate.instant('workflow_updated'));
        });
    }
}
