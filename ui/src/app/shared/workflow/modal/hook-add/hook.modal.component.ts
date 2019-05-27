import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { WNode, WNodeHook, Workflow } from '../../../../model/workflow.model';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { WorkflowNodeHookFormComponent } from '../../wizard/hook/hook.form.component';

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

    @Output() hookEvent = new EventEmitter<WNodeHook>();
    @Output() deleteHookEvent = new EventEmitter<WNodeHook>();

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
        this.deleteHookEvent.emit(this.hook);
    }

    saveHook(): void {
        let updatedHook = this.hookFormComponent.hook;
        this.hookEvent.emit(updatedHook);
    }
}
