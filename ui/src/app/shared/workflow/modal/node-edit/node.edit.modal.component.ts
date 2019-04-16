import {Component, Input, ViewChild} from '@angular/core';
import {Project} from 'app/model/project.model';
import {WNode, Workflow} from 'app/model/workflow.model';
import {ModalSize, ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-node-edit-modal',
    templateUrl: './node.edit.modal.html',
    styleUrls: ['./node.edit.modal.scss']
})
export class WorkflowNodeEditModalComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;

    @ViewChild('nodeEditModal')
    public nodeEditModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    selected = 'context';

    constructor(private _modalService: SuiModalService) {
    }

    show(): void {
        if (this.nodeEditModal) {
            const modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeEditModal);
            modalConfig.mustScroll = true;
            modalConfig.size = ModalSize.Large;
            this.modal = this._modalService.open(modalConfig);
        }
    }
}
