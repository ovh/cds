import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Project} from '../../../../model/project.model';
import {WNode, Workflow} from '../../../../model/workflow.model';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-node-outgoinghook-modal',
    templateUrl: './outgoing.modal.html',
    styleUrls: ['./outgoing.modal.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeOutGoingHookEditComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;
    @Input() loading: boolean;

    @Output() outgoingHookEvent = new EventEmitter<Workflow>();

    @ViewChild('nodeOutgoinghookModal')
    public nodeOutgoinghookModal: ModalTemplate<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;


    editableNode: WNode;

    constructor(private _modalService: SuiModalService) {
    }

    show(): void {
        if (this.nodeOutgoinghookModal) {
            this.editableNode = cloneDeep(this.node);
            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeOutgoinghookModal);
            this.modalConfig.mustScroll = true;
            this.modal = this._modalService.open(this.modalConfig);
        }
    }

    saveOutGoingHook(): void {
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let node = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        node.outgoing_hook = cloneDeep(this.editableNode.outgoing_hook);
        this.outgoingHookEvent.emit(clonedWorkflow);
    }
}
