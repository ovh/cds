import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { ModalTemplate, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal, SuiModalService } from 'ng2-semantic-ui/dist';
import { Observable, Subscription } from 'rxjs';
import { finalize } from 'rxjs/internal/operators/finalize';
import {
    InstanceStatus,
    InstanceStatusUtil,
    OperationStatus,
    OperationStatusUtil,
    ParamData,
    WorkflowTemplate,
    WorkflowTemplateBulk,
    WorkflowTemplateBulkOperation,
    WorkflowTemplateInstance
} from '../../../model/workflow-template.model';
import { WorkflowTemplateService } from '../../../service/services.module';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { Column, ColumnType, Select } from '../../../shared/table/data-table.component';

@Component({
    selector: 'app-workflow-template-bulk-modal',
    templateUrl: './workflow-template.bulk-modal.html',
    styleUrls: ['./workflow-template.bulk-modal.scss']
})
@AutoUnsubscribe()
export class WorkflowTemplateBulkModalComponent {
    @ViewChild('workflowTemplateBulkModal') workflowTemplateBulkModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    open: boolean;

    @Input() workflowTemplate: WorkflowTemplate;
    @Output() close = new EventEmitter();
    columnsInstances: Array<Column<WorkflowTemplateInstance>>;
    columnsOperations: Array<Column<WorkflowTemplateBulkOperation>>;
    instances: Array<WorkflowTemplateInstance>;
    loadingInstances: boolean;
    currentStep = 0;
    selectedInstanceKeys: Array<string> = new Array<string>();
    selectedInstances: Array<WorkflowTemplateInstance>;
    accordionOpenedIndex = 0;
    parameters: { [s: number]: ParamData };
    response: WorkflowTemplateBulk;
    pollingStatusSub: Subscription;

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

        this.columnsOperations = [
            <Column<WorkflowTemplateBulkOperation>>{
                name: 'common_workflow',
                selector: (i: WorkflowTemplateBulkOperation) => i.request.project_key + '/' + i.request.workflow_name
            }, <Column<WorkflowTemplateBulkOperation>>{
                name: '',
                selector: (i: WorkflowTemplateBulkOperation) => i.error
            }, <Column<WorkflowTemplateBulkOperation>>{
                type: ColumnType.LABEL,
                name: 'common_status',
                class: 'right aligned',
                selector: (i: WorkflowTemplateBulkOperation) => {
                    return {
                        class: OperationStatusUtil.color(i.status),
                        value: OperationStatusUtil.translate(i.status)
                    };
                }
            }
        ];

        this.parameters = {};
    }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.workflowTemplateBulkModal);

        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(() => {
            this.open = false;
            this.close.emit();
        });
        this.modal.onDeny(() => {
            this.open = false;
            this.close.emit();
        });

        this.clickGoToInstanceReset();
    }

    clickClose() {
        this.modal.approve(true);
    }

    selectFunc: Select<WorkflowTemplateInstance> = (d: WorkflowTemplateInstance): boolean => {
        if (!this.selectedInstanceKeys || this.selectedInstanceKeys.length === 0) {
            return d.status(this.workflowTemplate) === InstanceStatus.NOT_UP_TO_DATE;
        }
        return !!this.selectedInstanceKeys.find(k => k === d.key());
    }

    selectChange(e: Array<string>) {
        this.selectedInstanceKeys = e;
    }

    moveToStep(n: number) {
        if (this.currentStep !== n && this.currentStep === 2) {
            this.pollingStatusSub.unsubscribe();
        }
        this.currentStep = n;
    }

    clickGoToInstance() {
        this.moveToStep(0);
    }

    clickGoToInstanceReset() {
        this.loadingInstances = true;
        this._workflowTemplateService.getInstances(this.workflowTemplate.group.name, this.workflowTemplate.slug)
            .pipe(finalize(() => this.loadingInstances = false))
            .subscribe(is => this.instances = is.sort((a, b) => a.key() < b.key() ? -1 : 1));

        this.selectedInstanceKeys = [];

        this.clickGoToInstance();
    }

    clickGoToParam() {
        this.selectedInstances = this.instances.filter(i => !!this.selectedInstanceKeys.find(k => k === i.key()));
        this.moveToStep(1);
    }

    clickRunBulk() {
        let req = new WorkflowTemplateBulk();

        req.operations = this.selectedInstances.map(i => {
            let operation = new WorkflowTemplateBulkOperation();
            operation.request = i.request;
            if (this.parameters[i.id]) {
                operation.request.parameters = this.parameters[i.id];
            }
            return operation;
        });

        this.response = null;
        this._workflowTemplateService.bulk(this.workflowTemplate.group.name, this.workflowTemplate.slug, req).subscribe(b => {
            this.response = b;
            this.startPollingStatus();
        });

        this.moveToStep(2);
    }

    accordionOpen(e: any, index: number) {
        if (this.accordionOpenedIndex === index) {
            this.accordionOpenedIndex = -1; // close all accordion items
            return;
        }
        this.accordionOpenedIndex = index;
    }

    changeParam(instanceID: number, params: ParamData) {
        this.parameters[instanceID] = params;
    }

    startPollingStatus() {
        this.pollingStatusSub = Observable.interval(500).subscribe(() => {
            this._workflowTemplateService.getBulk(this.workflowTemplate.group.name,
                this.workflowTemplate.slug, this.response.id).subscribe(b => {
                    this.response = b;

                    // check if all operation are done to stop polling
                    let done = true;
                    for (let i = 0; i < this.response.operations.length; i++) {
                        let o = this.response.operations[i];
                        if (o.status !== OperationStatus.DONE && o.status !== OperationStatus.ERROR) {
                            done = false;
                            break;
                        }
                    }
                    if (done) {
                        this.pollingStatusSub.unsubscribe();
                    }
                })
        });
    }
}
