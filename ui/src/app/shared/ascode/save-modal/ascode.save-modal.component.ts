import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Operation } from 'app/model/operation.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { Observable, Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { ParamData } from '../save-form/ascode.save-form.component';

@Component({
    selector: 'app-ascode-save-modal',
    templateUrl: './ascode.save-modal.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class AsCodeSaveModalComponent {
    @ViewChild('updateAsCodeModal')
    public myModalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() name: string;

    dataToSave: any;
    dataType: string;
    loading: boolean;
    webworkerSub: Subscription;
    asCodeOperation: Operation;
    pollingOperationSub: Subscription;
    parameters: ParamData;
    repositoryFullname: string;

    constructor(
        private _modalService: SuiModalService,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _workflowService: WorkflowService,
        private _pipService: PipelineService
    ) { }

    show(data: any, type: string) {
        this.loading = false;
        this.dataToSave = data;
        this.dataType = type;

        if (this.workflow && this.workflow.workflow_data.node.context) {
            let rootAppID = this.workflow.workflow_data.node.context.application_id;
            if (rootAppID) {
                let rootApp = this.workflow.applications[rootAppID];
                if (rootApp.repository_fullname) {
                    this.repositoryFullname = rootApp.repository_fullname;
                }
            }
        }

        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.myModalTemplate);
        this.modal = this._modalService.open(this.modalConfig);
    }

    close() {
        this.modal.approve(true);
    }

    save(): void {
        switch (this.dataType) {
            case 'workflow':
                this.loading = true;
                this._cd.markForCheck();
                this._workflowService.updateAsCode(this.project.key, this.name, this.parameters.branch_name,
                    this.parameters.commit_message, this.dataToSave).subscribe(o => {
                        this.asCodeOperation = o;
                        this.startPollingOperation(this.name);
                    });
                break;
            case 'pipeline':
                this.loading = true;
                this._cd.markForCheck();
                this._pipService.updateAsCode(this.project.key, <Pipeline>this.dataToSave,
                    this.parameters.branch_name, this.parameters.commit_message).subscribe(o => {
                        this.asCodeOperation = o;
                        this.startPollingOperation((<Pipeline>this.dataToSave).workflow_ascode_holder.name);
                    });
                break;
            default:
                this._toast.error('', this._translate.instant('ascode_error_unknown_type'))
        }
    }

    startPollingOperation(workflowName: string) {
        this.pollingOperationSub = Observable.interval(1000)
            .mergeMap(_ => this._workflowService.getAsCodeOperation(this.project.key, workflowName, this.asCodeOperation.uuid))
            .first(o => o.status > 1)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(o => {
                this.asCodeOperation = o;
            });
    }

    onParamChange(param: ParamData): void {
        this.parameters = param;
    }
}
