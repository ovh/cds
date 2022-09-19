import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { EventService } from 'app/event.service';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { EventType } from 'app/model/event.model';
import { Operation } from 'app/model/operation.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { ApplicationService } from 'app/service/application/application.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { EventState } from 'app/store/event.state';
import { Subscription } from 'rxjs';
import { filter, finalize, first, map } from 'rxjs/operators';
import { ParamData } from '../save-form/ascode.save-form.component';
import { NzModalRef } from 'ng-zorro-antd/modal';

@Component({
    selector: 'app-ascode-save-modal',
    templateUrl: './ascode.save-modal.html',
    styleUrls: ['./ascode.save-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class AsCodeSaveModalComponent implements OnDestroy {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() name: string;

    dataToSave: any;
    dataType: string;
    loading: boolean;
    asCodeOperation: Operation;
    pollingOperationSub: Subscription;
    parameters: ParamData;
    repositoryFullname: string;
    canSave = false;
    displayCloseButton = false;

    constructor(
        private _modal: NzModalRef,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _workflowService: WorkflowService,
        private _pipService: PipelineService,
        private _appService: ApplicationService,
        private _store: Store,
        private _eventService: EventService,
        private _envService: EnvironmentService
    ) { }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        if (this.workflow && this.workflow.workflow_data.node.context) {
            let rootAppID = this.workflow.workflow_data.node.context.application_id;
            if (rootAppID) {
                let rootApp = this.workflow.applications[rootAppID];
                if (rootApp.repository_fullname) {
                    this.repositoryFullname = rootApp.repository_fullname;
                }
            }
        }
    }

    close() {
        delete this.parameters;
        this._modal.destroy();
    }

    save(): void {
        switch (this.dataType) {
            case 'workflow':
                this.loading = true;
                this._cd.markForCheck();
                this._workflowService.updateAsCode(this.project.key, this.name, this.parameters.branch_name,
                    this.parameters.commit_message, this.dataToSave as Workflow).subscribe(o => {
                        this.asCodeOperation = o;
                        this.startPollingOperation();
                    });
                break;
            case 'pipeline':
                this.loading = true;
                this._cd.markForCheck();
                this._pipService.updateAsCode(this.project.key, <Pipeline>this.dataToSave,
                    this.parameters.branch_name, this.parameters.commit_message).subscribe(o => {
                        this.asCodeOperation = o;
                        this.startPollingOperation();
                    });
                break;
            case 'application':
                this.loading = true;
                this._cd.markForCheck();
                this._appService.updateAsCode(this.project.key, this.name, <Application>this.dataToSave,
                    this.parameters.branch_name, this.parameters.commit_message).subscribe(o => {
                        this.asCodeOperation = o;
                        this.startPollingOperation();
                    });
                break;
            case 'environment':
                this.loading = true;
                this._cd.markForCheck();
                this._envService.updateAsCode(this.project.key, this.name, <Environment>this.dataToSave,
                    this.parameters.branch_name, this.parameters.commit_message).subscribe(o => {
                    this.asCodeOperation = o;
                    this.startPollingOperation();
                });
                break;
            default:
                this._toast.error('', this._translate.instant('ascode_error_unknown_type'));
        }
    }

    startPollingOperation() {
        this.pollingOperationSub = this._store.select(EventState.last)
            .pipe(
                filter(e => e && e.type_event === EventType.OPERATION && e.project_key === this.project.key),
                map(e => e.payload as Operation),
                filter(o => o.uuid === this.asCodeOperation.uuid),
                first(o => o.status > 1),
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(o => {
                this.asCodeOperation = o;
                if (this.asCodeOperation.status === 2) {
                    this.displayCloseButton = true;
                }

            });
        this._eventService.subscribeToOperation(this.project.key, this.asCodeOperation.uuid);
    }

    onParamChange(param: ParamData): void {
        this.parameters = param;
        this.canSave = !this.isEmpty(this.parameters.commit_message) && !this.isEmpty(this.parameters.branch_name);
        this._cd.markForCheck();
    }

    isEmpty(str: string): boolean {
        return (!str || str.length === 0);
    }
}
