import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { cloneDeep } from 'lodash';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { WNode, WNodeHook, Workflow } from '../../../../model/workflow.model';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { WorkflowNodeHookFormComponent } from '../../node/hook/form/hook.form.component';

@Component({
    selector: 'app-hook-modal',
    templateUrl: './hook.modal.html',
    styleUrls: ['./hook.modal.scss']
})
@AutoUnsubscribe()
export class WorkflowHookModalComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;
    @Input() loading: boolean;

    @Input() hook: WNodeHook;

    @Output() hookEvent = new EventEmitter<Workflow>();

    @ViewChild('hookModalComponent')
    public hookModalComponent: ModalTemplate<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @ViewChild('hookFormComponent')
    hookFormComponent: WorkflowNodeHookFormComponent;

    permissionEnum = PermissionValue;

    constructor(private _modalService: SuiModalService) {
    }

    show(): void {
        if (this.hookModalComponent) {
            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.hookModalComponent);
            this.modalConfig.mustScroll = true;
            this.modal = this._modalService.open(this.modalConfig);
        }
    }

    deleteHook(): void {
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let clonedNode = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        clonedNode.hooks = clonedNode.hooks.filter(h => h.uuid !== this.hook.uuid);
        this.hookEvent.emit(clonedWorkflow);
    }

    saveHook(): void {
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let clonedNode = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        let updatedHook = this.hookFormComponent.hook;
        if (updatedHook.uuid) {
            // Update hook
            let existingHook = clonedNode.hooks.find(h => h.uuid === updatedHook.uuid);
            if (existingHook) {
                existingHook.config = updatedHook.config;
            }
        } else {
            // insert new hook
            if (!clonedNode.hooks) {
                clonedNode.hooks = new Array<WNodeHook>();
            }
            clonedNode.hooks.push(updatedHook);
        }
        this.hookEvent.emit(clonedWorkflow);
    }
}
