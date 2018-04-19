import {AfterViewInit, Component, ElementRef, Input, Output, EventEmitter, NgZone, ViewChild, OnInit} from '@angular/core';
import {Router} from '@angular/router';
import {Workflow, WorkflowNodeJoin} from '../../../model/workflow.model';
import {WorkflowRun, WorkflowRunRequest, WorkflowNodeRunManual} from '../../../model/workflow.run.model';
import {PipelineStatus} from '../../../model/pipeline.model';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';
import {WorkflowDeleteJoinComponent} from './delete/workflow.join.delete.component';
import {WorkflowRunService} from '../../../service/workflow/run/workflow.run.service';
import {Project} from '../../../model/project.model';
import {WorkflowTriggerJoinComponent} from './trigger/trigger.join.component';
import {Subscription} from 'rxjs/Subscription';
import {finalize} from 'rxjs/operators';
import {WorkflowEventStore} from '../../../service/workflow/workflow.event.store';

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
    @Output() selectEvent = new EventEmitter<WorkflowNodeJoin>();

    readonly = false;
    isSelected = false;
    subSelect: Subscription;
    disabled = false;
    loading = false;

    @ViewChild('workflowDeleteJoin')
    workflowDeleteJoin: WorkflowDeleteJoinComponent;
    @ViewChild('workflowJoinTrigger')
    workflowJoinTrigger: WorkflowTriggerJoinComponent;

    pipelineStatusEnum = PipelineStatus;

    zone: NgZone;
    options: {};

    workflowCoreSub: Subscription;
    currentWorkflowRun: WorkflowRun;

    constructor(private elementRef: ElementRef, private _workflowRunService: WorkflowRunService,
        private _router: Router,
        private _workflowEventStore: WorkflowEventStore) {
        this.zone = new NgZone({enableLongStackTrace: false});
        this.options = {
            'fullTextSearch': true,
            onHide: () => {
                this.zone.run(() => {
                    this.elementRef.nativeElement.style.zIndex = 0;
                });
            }
        };

        this.subSelect = this._workflowEventStore.selectedJoin().subscribe(j => {
            if (j && this.join) {
                this.isSelected = this.join.id === j.id;
                return;
            }
            this.isSelected = false;
        });

        this.workflowCoreSub = this._workflowEventStore.selectedRun().subscribe(wr => {
            this.currentWorkflowRun = wr;
            this.readonly = this.currentWorkflowRun != null;
        });
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = 0;
    }

    ngOnInit() {
    }

    selectJoinToLink(): void {
        this.selectEvent.emit(this.join);
    }

    selectJoin(): void {
        this._workflowEventStore.setSelectedJoin(this.join);
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
                this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', wr.num],
                {queryParams: { subnum: wr.last_subnumber }});
            });
    }
}
