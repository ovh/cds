import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    Output
} from '@angular/core';
import { Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { EventService } from 'app/event.service';
import { EventType } from 'app/model/event.model';
import { Operation } from 'app/model/operation.model';
import { Project } from 'app/model/project.model';
import {
    ParamData,
    WorkflowTemplate,
    WorkflowTemplateApplyResult,
    WorkflowTemplateInstance,
    WorkflowTemplateRequest
} from 'app/model/workflow-template.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowTemplateService } from 'app/service/workflow-template/workflow-template.service';
import { ParamData as AsCodeParamData } from 'app/shared/ascode/save-form/ascode.save-form.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { EventState } from 'app/store/event.state';
import { Subscription } from 'rxjs';
import { filter, finalize, first, map } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-template-apply-form',
    templateUrl: './workflow-template.apply-form.html',
    styleUrls: ['./workflow-template.apply-form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowTemplateApplyFormComponent implements OnChanges, OnDestroy {
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() workflowTemplate: WorkflowTemplate;
    @Input() workflowTemplateInstance: WorkflowTemplateInstance;
    @Input() withClose: boolean;
    @Output() close = new EventEmitter<number>();
    @Output() apply = new EventEmitter<number>();

    loading: boolean;
    result: WorkflowTemplateApplyResult;
    parameterName: string;
    parameterValues: ParamData;
    detached: boolean;
    asCodeOperation: Operation;
    pollingOperationSub: Subscription;
    asCodeApply: boolean;
    asCodeParameters: AsCodeParamData;
    validFields: boolean;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _router: Router,
        private _cd: ChangeDetectorRef,
        private _eventService: EventService,
        private _store: Store
    ) { }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnChanges() {
        this.parameterName = this.workflowTemplateInstance ? this.workflowTemplateInstance.request.workflow_name : '';
        this.asCodeApply = this.workflow && !!this.workflow.from_repository;
    }

    applyTemplate() {
        let req = <WorkflowTemplateRequest>{
            project_key: this.project.key,
            workflow_name: this.parameterName,
            parameters: this.parameterValues,
            detached: !!this.detached
        };

        this.result = null;
        this.loading = true;
        this._cd.markForCheck();

        if (this.asCodeApply) {
            this._workflowTemplateService.applyAsCode(this.workflowTemplate.group.name, this.workflowTemplate.slug, req,
                this.asCodeParameters.branch_name, this.asCodeParameters.commit_message)
                .subscribe(o => {
                    this.asCodeOperation = o;
                    this.startPollingOperation();
                });
            return;
        }

        this._workflowTemplateService.apply(this.workflowTemplate.group.name, this.workflowTemplate.slug, req)
            .pipe(first(), finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                // if the workflow name changed move to new workflow page
                this.result = res;

                // specific check for case where workflow name change in template
                if (this.workflow && res.workflow_name !== this.workflow.name) {
                    this._router.navigate(['/project', this.project.key, 'workflow', res.workflow_name]);
                } else {
                    this.apply.emit();
                }
            });
    }

    goToWorkflow(): void {
        this._router.navigate(['/project', this.project.key, 'workflow', this.result.workflow_name]);
    }

    filterRepo(options: Array<string>, query: string): Array<string> | false {
        if (!options) {
            return false;
        }
        if (!query || query.length < 3) {
            return options.slice(0, 100);
        }
        let queryLowerCase = query.toLowerCase();
        return options.filter(name => name.toLowerCase().indexOf(queryLowerCase) !== -1);
    }

    clickClose() {
        this.close.emit();
    }

    changeParam(values: { [key: string]: string; }) {
        this.parameterValues = values;
        this.validateParam();
    }

    clickDetach() {
        this._workflowTemplateService.deleteInstance(this.workflowTemplate, this.workflowTemplateInstance)
            .subscribe(() => {
                this.clickClose();
            });
    }

    onSelectDetachChange() {
        this.detached = !this.detached;
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
            });
        this._eventService.subscribeToOperation(this.project.key, this.asCodeOperation.uuid);
    }

    onAsCodeParamChange(param: AsCodeParamData): void {
        this.asCodeParameters = param;
        this.validateParam();
    }

    validateParam() {
        this.validFields = !this.asCodeApply || (this.asCodeParameters &&
            !!this.asCodeParameters.branch_name && !!this.asCodeParameters.commit_message);
    }
}
