import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {Store} from '@ngxs/store';
import {Project} from 'app/model/project.model';
import {WNode, Workflow} from 'app/model/workflow.model';
import {CleanWorkflowNodeModal} from 'app/store/node.modal.action';
import {NodeModalState, NodeModalStateModel} from 'app/store/node.modal.state';
import {ModalSize, ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-node-edit-modal',
    templateUrl: './node.edit.modal.html',
    styleUrls: ['./node.edit.modal.scss']
})
export class WorkflowNodeEditModalComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;

    @ViewChild('nodeEditModal')
    public nodeEditModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    selected = 'context';

    constructor(private _modalService: SuiModalService, private _store: Store) {
    }

    ngOnInit(): void {
        this._store.select(NodeModalState.getCurrent()).subscribe( (s: NodeModalStateModel) => {
            if (s.node) {
                this.project = s.project;
                this.workflow = s.workflow;
                this.node = s.node;
                if (!this.modal) {
                    this.show();
                }
            } else if (this.modal) {
                this.modal.approve(true);
            }
        });
    }

    show(): void {
        if (this.nodeEditModal) {
            const modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeEditModal);
            modalConfig.mustScroll = true;
            modalConfig.size = ModalSize.Large;
            this.modal = this._modalService.open(modalConfig);
            this.modal.onApprove(() => {
                this._store.dispatch(new CleanWorkflowNodeModal({}));
            });
            this.modal.onDeny(() => {
                this._store.dispatch(new CleanWorkflowNodeModal({}));
            });
        }
    }
}
