import {Component, Input, OnInit, ViewChild, ChangeDetectorRef, EventEmitter} from '@angular/core';
import {Router} from '@angular/router';
import {Project} from '../../../../../model/project.model';
import {
    Workflow,
    WorkflowNode,
    WorkflowNodeTrigger,
    WorkflowNodeHook,
    WorkflowNodeJoin,
    WorkflowPipelineNameImpact
} from '../../../../../model/workflow.model';
import {WorkflowRun, WorkflowNodeRun} from '../../../../../model/workflow.run.model';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {AuthentificationStore} from '../../../../../service/auth/authentification.store';
import {WorkflowTriggerComponent} from '../../../../../shared/workflow/trigger/workflow.trigger.component';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {WorkflowDeleteNodeComponent} from '../../../../../shared/workflow/node/delete/workflow.node.delete.component';
import {WorkflowNodeContextComponent} from '../../../../../shared/workflow/node/context/workflow.node.context.component';
import {PipelineStore} from '../../../../../service/pipeline/pipeline.store';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WorkflowNodeHookFormComponent} from '../../../../../shared/workflow/node/hook/form/node.hook.component';
import {HookEvent} from '../../../../../shared/workflow/node/hook/hook.event';
import {WorkflowNodeRunParamComponent} from '../../../../../shared/workflow/node/run/node.run.param.component';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {WorkflowCoreService} from '../../../../../service/workflow/workflow.core.service';
import {WorkflowNodeConditionsComponent} from '../../../../../shared/workflow/node/conditions/node.conditions.component';
import {Subscription} from 'rxjs/Subscription';
import {first} from 'rxjs/operators';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow-sidebar-run-node',
    templateUrl: './workflow.sidebar.run.node.component.html',
    styleUrls: ['./workflow.sidebar.run.node.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunNodeComponent implements OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WorkflowNode;
    // Flag indicate if sidebar is open
    @Input() runId: number;
    @Input() runNumber: number;
    @Input() number: number;
    @Input() open: boolean;

    // Modal
    @ViewChild('nodeParentModal')
    nodeParentModal: ModalTemplate<boolean, boolean, void>;
    @ViewChild('workflowRunNode')
    workflowRunNode: WorkflowNodeRunParamComponent;
    newParentNode: WorkflowNode;
    modalParentNode: ActiveModal<boolean, boolean, void>;
    newTrigger: WorkflowNodeTrigger = new WorkflowNodeTrigger();
    previousNodeName: string;
    pipelineSubscription: Subscription;
    displayInputName = false;
    loading = false;
    nameWarning: WorkflowPipelineNameImpact;
    currentWorkflowRun: WorkflowRun;
    currentWorkflowNodeRun: WorkflowNodeRun;
    pipelineStatusEnum = PipelineStatus;

    constructor(private _changeDetectorRef: ChangeDetectorRef,
                private _workflowStore: WorkflowStore, private _translate: TranslateService, private _toast: ToastService,
                private _wrService: WorkflowRunService, private _pipelineStore: PipelineStore, private _router: Router,
                private _modalService: SuiModalService, private _workflowCoreService: WorkflowCoreService) {

    }

    ngOnInit() {
        this._workflowCoreService.getCurrentWorkflowRun()
            .subscribe((wr) => {
                if (!wr) {
                    return;
                }
                this.currentWorkflowRun = wr;

                if (wr.nodes && wr.nodes[this.node.id] && Array.isArray(wr.nodes[this.node.id])) {
                    this.currentWorkflowNodeRun = wr.nodes[this.node.id].find((n) => n.id === this.runId && n.num === this.runNumber);
                }
                console.log(this.currentWorkflowNodeRun);
            });
    }

    displayLogs() {
        let pip = Workflow.getNodeByID(this.node.id, this.workflow).pipeline.name;
        this._router.navigate([
            '/project', this.project.key,
            'workflow', this.workflow.name,
            'run', this.runNumber,
            'node', this.runId], {queryParams: {name: pip}});
    }

    stopNodeRun(): void {
        this.loading = true;
        this._wrService.stopNodeRun(this.project.key, this.workflow.name, this.runNumber, this.runId)
            .pipe(first())
            .subscribe(() => {
                this._router.navigate([
                    '/project', this.project.key,
                    'workflow', this.workflow.name,
                    'run', this.runNumber]);
                // this._changeDetectorRef.detach();
                // setTimeout(() => this._changeDetectorRef.reattach(), 2000);
                // this._toast.success('', this._translate.instant('pipeline_stop'));
            });
    }

    openRunNode(): void {
        this.workflowRunNode.show();
    }
}
