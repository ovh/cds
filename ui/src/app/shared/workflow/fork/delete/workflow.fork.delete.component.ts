import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WorkflowNodeFork} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-fork-delete',
    templateUrl: './workflow.fork.delete.html',
    styleUrls: ['./workflow.fork.delete.scss']
})
export class WorkflowDeleteForkComponent implements OnInit {

    @ViewChild('deleteModal')
    deleteModalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Output() deleteEvent = new EventEmitter<string>();
    @Input() fork: WorkflowNodeFork;
    @Input() isRoot: boolean;
    @Input() loading: boolean;

    deleteAll = 'only';

    constructor(private _modalService: SuiModalService) {

    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.deleteModalTemplate);
        this.modal = this._modalService.open(config);
    }

    deleteFork(): void {
        this.deleteEvent.emit(this.deleteAll);
    }

    ngOnInit(): void {
    }
}
