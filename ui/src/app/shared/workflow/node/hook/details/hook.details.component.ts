import {Component, ViewChild} from '@angular/core';
import {TaskExecution} from '../../../../../model/workflow.hook.model';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {cloneDeep} from 'lodash';

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
            autoRefresh: true,
            readOnly: true
        };
    }

    show(taskExec: TaskExecution): void {
        this.task = cloneDeep(taskExec);
        if (this.task.webhook && this.task.webhook.request_body) {
          let body = atob(this.task.webhook.request_body);
          try {
            this.task.webhook.request_body = JSON.stringify(JSON.parse(body), null, 4);
          } catch (e) {
            this.task.webhook.request_body = body;
          }

        }
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeHookDetailsModal);
        this.modal = this._modalService.open(this.modalConfig);
    }
}
