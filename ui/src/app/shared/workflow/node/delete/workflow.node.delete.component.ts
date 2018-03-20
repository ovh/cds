import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {WorkflowNode} from '../../../../model/workflow.model';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-workflow-node-delete',
    templateUrl: './workflow.node.delete.html',
    styleUrls: ['./workflow.node.delete.scss']
})
export class WorkflowDeleteNodeComponent {

    @ViewChild('deleteModal')
    deleteModalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Output() deleteEvent = new EventEmitter<string>();
    @Input() node: WorkflowNode;
    @Input() isRoot: boolean;
    @Input() loading: boolean;

    deleteAll = 'only';

    constructor(private _modalService: SuiModalService) {

    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.deleteModalTemplate);
        this.modal = this._modalService.open(config);
    }

    deleteNode(): void {
        this.deleteEvent.emit(this.deleteAll);
    }
}
