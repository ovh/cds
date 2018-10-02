import {AfterViewInit, Component, ElementRef, Input, OnInit} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {WNode, WNodeJoin, Workflow} from '../../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../../model/workflow.run.model';
import {WorkflowCoreService} from '../../../../service/workflow/workflow.core.service';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {ToastService} from '../../../toast/ToastService';

@Component({
    selector: 'app-workflow-wnode-join',
    templateUrl: './node.join.html',
    styleUrls: ['./node.join.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeJoinComponent implements OnInit, AfterViewInit {

    @Input() public project: Project;
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public workflowrun: WorkflowRun;
    @Input() public selected: boolean;

    canRun: boolean;
    pipelineStatusEnum = PipelineStatus;

    elementRef: ElementRef;
    linkJoinSubscription: Subscription;
    nodeToLink: WNode;
    loading = false;
    loadingRun = false;

    constructor(elt: ElementRef, private _workflowCore: WorkflowCoreService, private _workflowStore: WorkflowStore,
                private _toast: ToastService, private _translate: TranslateService) {
        this.elementRef = elt;

        this.linkJoinSubscription = _workflowCore.getLinkJoinEvent().subscribe(n => {
            this.nodeToLink = n;
        });
    }

    ngOnInit(): void {
        this.canBeLaunched();
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
    }

    canBeLaunched() {
        if (!this.workflowrun || !this.workflowrun.nodes) {
            return false;
        }
        let lengthParentRun = 0;
        Object.keys(this.workflowrun.nodes).forEach((key) => {
            if (this.workflowrun.nodes[key].length &&
                this.node.parents.findIndex(p => p.parent_id === this.workflowrun.nodes[key][0].workflow_node_id) !== -1) {
                lengthParentRun++;
            }
        });
        this.canRun = this.node.parents.length === lengthParentRun;
    }

    selectJoinToLink(): void {
        let cloneWorkflow = cloneDeep(this.workflow);
        let currentJoin = Workflow.getNodeByID(this.node.id, cloneWorkflow);
        if (currentJoin.parents.findIndex(p => p.parent_name === this.nodeToLink.ref) === -1) {
            let joinParent = new WNodeJoin();
            joinParent.parent_name = this.nodeToLink.ref;
            currentJoin.parents.push(joinParent);
        }
        this._workflowCore.linkJoinEvent(null);
        this.updateWorkflow(cloneWorkflow);
    }

    updateWorkflow(w: Workflow): void {
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
        });
    }

    playJoin(): void {
        // TODO
        /*
        let request = new WorkflowRunRequest();
        if (!this.node || !Array.isArray(this.node.triggers)) {
            return;
        }
        this.loading = true;
        this.loadingRun = true;

        request.manual = new WorkflowNodeRunManual();
        request.from_nodes = this.join.triggers.map((trig) => trig.workflow_dest_node_id);
        request.number = this.currentWorkflowRun.num;

        this._workflowRunService.runWorkflow(this.project.key, this.workflow.name, request)
            .pipe(finalize(() => {
                this.loading = false;
                this.loadingRun = false;
            }))
            .subscribe((wr) => {
                this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', wr.num],
                    {queryParams: { subnum: wr.last_subnumber }});
            });
            */
    }
}
