import {Component, Input, ViewChild} from '@angular/core';
import {HookService} from '../../../../../service/hook/hook.service';
import {TaskExecution} from '../../../../../model/workflow.hook.model';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-workflow-node-hook-details',
    templateUrl: './hook.details.component.html',
    styleUrls: ['./hook.details.component.scss']
})
export class WorkflowNodeHookDetailsComponent {
    codeMirrorConfig: any;

    // Ng semantic modal
    @ViewChild('nodeHookDetailsModal')
    public nodeHookDetailsModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    task: TaskExecution;

    constructor(private _modalService: SuiModalService) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true
        };
    }

    show(taskExec: TaskExecution): void {
        this.task = taskExec;
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeHookDetailsModal);
        this.modal = this._modalService.open(this.modalConfig);
    }
}
