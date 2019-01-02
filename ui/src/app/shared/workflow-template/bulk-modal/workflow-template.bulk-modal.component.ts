import { Component, Input, ViewChild } from '@angular/core';
import { ModalTemplate, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal, SuiModalService } from 'ng2-semantic-ui/dist';
import { finalize } from 'rxjs/internal/operators/finalize';
import { InstanceStatus, InstanceStatusUtil, WorkflowTemplate, WorkflowTemplateInstance } from '../../../model/workflow-template.model';
import { WorkflowTemplateService } from '../../../service/services.module';
import { Column, ColumnType, Select } from '../../../shared/table/data-table.component';

@Component({
    selector: 'app-workflow-template-bulk-modal',
    templateUrl: './workflow-template.bulk-modal.html',
    styleUrls: ['./workflow-template.bulk-modal.scss']
})
export class WorkflowTemplateBulkModalComponent {
    @ViewChild('workflowTemplateBulkModal') workflowTemplateBulkModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    open: boolean;

    @Input() workflowTemplate: WorkflowTemplate;
    columnsInstances: Array<Column<WorkflowTemplateInstance>>;
    instances: Array<WorkflowTemplateInstance>;
    loadingInstances: boolean;

    constructor(
        private _modalService: SuiModalService,
        private _workflowTemplateService: WorkflowTemplateService
    ) {
        this.columnsInstances = [
            <Column<WorkflowTemplateInstance>>{
                name: 'common_workflow',
                selector: (i: WorkflowTemplateInstance) => i.key()
            }, <Column<WorkflowTemplateInstance>>{
                type: ColumnType.LABEL,
                name: 'common_status',
                class: 'right aligned',
                selector: (i: WorkflowTemplateInstance) => {
                    let status = i.status(this.workflowTemplate);
                    return {
                        class: InstanceStatusUtil.color(status),
                        value: status
                    };
                }
            }
        ];
    }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.workflowTemplateBulkModal);

        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(() => { this.open = false; });
        this.modal.onDeny(() => { this.open = false; });

        this.loadingInstances = true;
        this._workflowTemplateService.getInstances(this.workflowTemplate.group.name, this.workflowTemplate.slug)
            .pipe(finalize(() => this.loadingInstances = false))
            .subscribe(is => { this.instances = is; });
    }

    close() {
        this.modal.approve(true);
    }

    selectFunc: Select<WorkflowTemplateInstance> = (d: WorkflowTemplateInstance): boolean => {
        return d.status(this.workflowTemplate) === InstanceStatus.NOT_UP_TO_DATE;
    }

    selectChange(e: Array<string>) {
        console.log(e);
    }
}
