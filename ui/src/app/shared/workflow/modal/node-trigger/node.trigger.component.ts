import {Component, Input, ViewChild} from '@angular/core';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Project} from '@cds/model/project.model';
import {WNode, Workflow} from '@cds/model/workflow.model';

@Component({
    selector: 'app-workflow-node-trigger',
    templateUrl: './node.trigger.component.html',
    styleUrls: ['./node.trigger.scss']
})
export class WorkflowNodeTriggerComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;

    @ViewChild('triggerModal')
    triggerModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    selected = 'pipeline';
    node = new WNode();

    constructor(private _modalService: SuiModalService){

    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.triggerModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
    }
}
