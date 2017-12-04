import {AfterViewInit, Component, ElementRef, EventEmitter, Input, NgZone, Output, ViewChild, OnInit} from '@angular/core';
import {Router} from '@angular/router';
import {Workflow, WorkflowNodeJoin, WorkflowNodeJoinTrigger} from '../../../model/workflow.model';
import {WorkflowRun, WorkflowRunRequest, WorkflowNodeRunManual} from '../../../model/workflow.run.model';
import {PipelineStatus} from '../../../model/pipeline.model';
import {cloneDeep} from 'lodash';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';
import {WorkflowDeleteJoinComponent} from './delete/workflow.join.delete.component';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {WorkflowRunService} from '../../../service/workflow/run/workflow.run.service';
import {Project} from '../../../model/project.model';
import {ToastService} from '../../toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {WorkflowTriggerJoinComponent} from './trigger/trigger.join.component';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {Subscription} from 'rxjs/Subscription';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-join',
    templateUrl: './workflow.join.html',
    styleUrls: ['./workflow.join.scss']
})
@AutoUnsubscribe()
export class WorkflowJoinComponent implements AfterViewInit, OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() join: WorkflowNodeJoin;
    @Input() readonly = false;

    disabled = false;
    loading = false;

    @ViewChild('workflowDeleteJoin')
    workflowDeleteJoin: WorkflowDeleteJoinComponent;
    @ViewChild('workflowJoinTrigger')
    workflowJoinTrigger: WorkflowTriggerJoinComponent;

    @Output() selectEvent = new EventEmitter<WorkflowNodeJoin>();

    newTrigger = new WorkflowNodeJoinTrigger();
    pipelineStatusEnum = PipelineStatus;

    zone: NgZone;
    options: {};

    workflowCoreSub: Subscription;
    currentWorkflowRun: WorkflowRun;

    constructor(private elementRef: ElementRef, private _workflowStore: WorkflowStore, private _toast: ToastService,
        private _translate: TranslateService, private _workflowRunService: WorkflowRunService,
        private _workflowCoreService: WorkflowCoreService, private _router: Router) {
        this.zone = new NgZone({enableLongStackTrace: false});
        this.options = {
            'fullTextSearch': true,
            onHide: () => {
                this.zone.run(() => {
                    this.elementRef.nativeElement.style.zIndex = 0;
                });
            }
        };
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = 0;
    }

    ngOnInit() {
        this.workflowCoreSub = this._workflowCoreService.getCurrentWorkflowRun().subscribe(wr => {
            this.currentWorkflowRun = wr;
        });
    }

    displayDropdown(): void {
        this.elementRef.nativeElement.style.zIndex = 50;
    }

    openDeleteJoinModal(): void {
        if (this.workflowDeleteJoin) {
            this.workflowDeleteJoin.show();
        }
    }

    openTriggerJoinModal(): void {
        this.newTrigger = new WorkflowNodeJoinTrigger();
        if (this.workflowJoinTrigger) {
            this.workflowJoinTrigger.show();
        }
    }

    deleteJoin(b: boolean): void {
        if (b) {
            let clonedWorkflow: Workflow = cloneDeep(this.workflow);
            clonedWorkflow.joins = clonedWorkflow.joins.filter(j => j.id !== this.join.id);
            Workflow.removeOldRef(clonedWorkflow);
            this.updateWorkflow(clonedWorkflow, this.workflowDeleteJoin.modal);
        }
    }

    updateWorkflow(w: Workflow, modal?: ActiveModal<boolean, boolean, void>): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('workflow_updated'));
            if (modal) {
                modal.approve(true);
            }
        }, () => {
            this.loading = false;
        });
    }

    saveTrigger(): void {
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let currentJoin: WorkflowNodeJoin = clonedWorkflow.joins.find(j => j.id === this.join.id);
        if (!currentJoin) {
            return;
        }

        if (!currentJoin.triggers) {
            currentJoin.triggers = new Array<WorkflowNodeJoinTrigger>();
        }
        currentJoin.triggers.push(cloneDeep(this.newTrigger));
        this.updateWorkflow(clonedWorkflow, this.workflowJoinTrigger.modal);
    }

    selectJoin(): void {
        this.selectEvent.emit(this.join);
    }

    canBeLaunched() {
        if (!this.currentWorkflowRun || !this.currentWorkflowRun.nodes) {
            return false;
        }
        let lengthParentRun = 0;
        Object.keys(this.currentWorkflowRun.nodes).forEach((key) => {
            if (this.currentWorkflowRun.nodes[key].length &&
              this.join.source_node_id.includes(this.currentWorkflowRun.nodes[key][0].workflow_node_id)) {
                lengthParentRun++;
            }
        });

        return this.join.source_node_id.length === lengthParentRun;
    }

    playJoin(): void {
        let request = new WorkflowRunRequest();
        if (!this.join || !Array.isArray(this.join.triggers)) {
            return;
        }
        this.loading = true;

        request.manual = new WorkflowNodeRunManual();
        request.from_nodes = this.join.triggers.map((trig) => trig.workflow_dest_node_id);
        request.number = this.currentWorkflowRun.num;

        this._workflowRunService.runWorkflow(this.project.key, this.workflow.name, request)
            .pipe(finalize(() => this.loading = false))
            .subscribe((wr) => {
                this._workflowCoreService.setCurrentWorkflowRun(wr);
                this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', wr.num],
                {queryParams: { subnum: wr.last_subnumber }});
            });
    }
}
