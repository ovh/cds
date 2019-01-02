import { Component, ViewChild } from '@angular/core';
import { ModalTemplate, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal, SuiModalService } from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-workflow-template-bulk-modal',
    templateUrl: './workflow-template.bulk-modal.html',
    styleUrls: ['./workflow-template.bulk-modal.scss']
})
export class WorkflowTemplateBulkModalComponent {
    @ViewChild('workflowTemplateBulkModal') workflowTemplateBulkModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    constructor(
        private _modalService: SuiModalService,
    ) { }

    show() {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.workflowTemplateBulkModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
    }

    close() {
        this.modal.approve(true);
    }
}
