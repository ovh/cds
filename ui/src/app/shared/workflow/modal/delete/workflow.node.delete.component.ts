import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { WNode, Workflow } from 'app/model/workflow.model';
import cloneDeep from 'lodash-es/cloneDeep';

@Component({
    selector: 'app-workflow-node-delete',
    templateUrl: './workflow.node.delete.html',
    styleUrls: ['./workflow.node.delete.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowDeleteNodeComponent {

    @ViewChild('deleteModal', { static: false })
    deleteModalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;

    @Input() node: WNode;
    @Input() workflow: Workflow;
    @Input() loading: boolean;
    @Output() deleteEvent = new EventEmitter<Workflow>();

    deleteAll = 'only';

    constructor(private _modalService: SuiModalService) { }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.deleteModalTemplate);
        this.modal = this._modalService.open(config);
    }

    deleteNode(): void {
        let clonedWorkflow = cloneDeep(this.workflow);
        if (this.deleteAll === 'only') {
            Workflow.removeNodeOnly(clonedWorkflow, this.node.id);
        } else {
            Workflow.removeNodeWithChild(clonedWorkflow, this.node.id);
        }
        this.deleteEvent.emit(clonedWorkflow);
    }
}
