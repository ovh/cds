import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WNode, Workflow} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-node-delete',
    templateUrl: './workflow.node.delete.html',
    styleUrls: ['./workflow.node.delete.scss']
})
export class WorkflowDeleteNodeComponent {

    @ViewChild('deleteModal')
    deleteModalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Input() node: WNode;
    @Input() workflow: Workflow;
    @Input() loading: boolean;
    @Output() deleteEvent = new EventEmitter<Workflow>();

    deleteAll = 'only';

    constructor(private _modalService: SuiModalService) {}

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
