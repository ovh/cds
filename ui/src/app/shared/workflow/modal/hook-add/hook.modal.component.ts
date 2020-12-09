import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnDestroy, Output, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Project } from 'app/model/project.model';
import { WNode, WNodeHook, Workflow } from 'app/model/workflow.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowNodeHookFormComponent } from 'app/shared/workflow/wizard/hook/hook.form.component';
import { WorkflowState } from 'app/store/workflow.state';
@Component({
    selector: 'app-hook-modal',
    templateUrl: './hook.modal.html',
    styleUrls: ['./hook.modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowHookModalComponent implements OnDestroy {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;
    @Input() loading: boolean;

    @Output() hookEvent = new EventEmitter<WNodeHook>();
    @Output() deleteHookEvent = new EventEmitter<WNodeHook>();

    editMode: boolean;

    @ViewChild('hookModalComponent')
    public hookModalComponent: ModalTemplate<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;

    @ViewChild('hookFormComponent')
    hookFormComponent: WorkflowNodeHookFormComponent;

    constructor(private _modalService: SuiModalService, private _store: Store, private _cd: ChangeDetectorRef) {
        this.editMode = this._store.selectSnapshot(WorkflowState).editMode;
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    show(): void {
        if (this.hookModalComponent) {
            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.hookModalComponent);
            this.modalConfig.mustScroll = true;
            this.modal = this._modalService.open(this.modalConfig);
            this._cd.detectChanges();
        }
    }

    saveHook(): void {
        let updatedHook = this.hookFormComponent.hook;
        this.hookEvent.emit(updatedHook);
        this.modal.approve(true);
    }
}
