import {
    AfterViewInit,
    ChangeDetectorRef,
    Component,
    ElementRef,
    EventEmitter,
    Input,
    NgZone,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import {
    Workflow,
    WorkflowNode,
    WorkflowNodeHook,
    WorkflowNodeJoin,
    WorkflowNodeTrigger,
    WorkflowPipelineNameImpact
} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';
import {WorkflowTriggerComponent} from '../trigger/workflow.trigger.component';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../toast/ToastService';
import {WorkflowDeleteNodeComponent} from './delete/workflow.node.delete.component';
import {WorkflowNodeContextComponent} from './context/workflow.node.context.component';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
import {Router, ActivatedRoute} from '@angular/router';
import {PipelineStatus} from '../../../model/pipeline.model';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WorkflowNodeHookFormComponent} from './hook/form/node.hook.component';
import {HookEvent} from './hook/hook.event';
import {WorkflowNodeRunParamComponent} from './run/node.run.param.component';
import {WorkflowRunService} from '../../../service/workflow/run/workflow.run.service';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {WorkflowNodeConditionsComponent} from './conditions/node.conditions.component';
import {first} from 'rxjs/operators';

declare var _: any;

@Component({
    selector: 'app-workflow-node',
    templateUrl: './workflow.node.html',
    styleUrls: ['./workflow.node.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeComponent implements AfterViewInit, OnInit {

    @Input() node: WorkflowNode;
    @Input() workflow: Workflow;
    @Input() project: Project;

    @ViewChild('workflowRunNode')
    workflowRunNode: WorkflowNodeRunParamComponent;

    workflowRun: WorkflowRun;
    workflowRunStatus: string;
    workflowRunNum: number;

    pipelineSubscription: Subscription;

    zone: NgZone;
    currentNodeRun: WorkflowNodeRun;
    pipelineStatus = PipelineStatus;


    loading = false;
    options: {};
    disabled = false;
    loadingStop = false;
    displayInputName = false;
    displayPencil = false;
    nameWarning: WorkflowPipelineNameImpact;
    selectedNodeId: number;

    workflowCoreSub: Subscription;

    constructor(private elementRef: ElementRef, private _changeDetectorRef: ChangeDetectorRef,
                private _workflowStore: WorkflowStore, private _translate: TranslateService, private _toast: ToastService,
                private _wrService: WorkflowRunService, private _pipelineStore: PipelineStore, private _router: Router,
                private _modalService: SuiModalService, private _workflowCoreService: WorkflowCoreService,
                private _route: ActivatedRoute) {
        this._route.queryParams.subscribe((qp) => {
            if (qp['selectedNodeId']) {
                this.selectedNodeId = parseInt(qp['selectedNodeId'], 10);
            } else {
                this.selectedNodeId = null;
            }
        });
    }

    ngOnInit(): void {
        this.zone = new NgZone({enableLongStackTrace: false});

        this.workflowCoreSub = this._workflowCoreService.getCurrentWorkflowRun().subscribe(wr => {
            if (wr) {
                if (this.workflowRun && this.workflowRun.id !== wr.id) {
                    this.currentNodeRun = null;
                }
                this.workflowRun = wr;
                if (wr.nodes && wr.nodes[this.node.id] && wr.nodes[this.node.id].length > 0) {
                    this.currentNodeRun = wr.nodes[this.node.id][0];
                }
            } else {
                this.workflowRun = null;
            }
        });
        if (!this.workflowRun) {
            this.options = {
                'fullTextSearch': true,
                onHide: () => {
                    this.zone.run(() => {
                        this.elementRef.nativeElement.style.zIndex = 0;
                    });
                }
            };
        }
    }

    goToNodeRun(): void {
        let qps = cloneDeep(this._route.snapshot.queryParams);
        qps['selectedJoinId'] = null;

        if (!this._route.snapshot.params['number']) {
            qps['selectedNodeRunId'] = null;
            qps['selectedNodeRunNum'] = null;

            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name
            ], { queryParams: Object.assign({}, qps, {selectedNodeId: this.node.id })});
        } else {
            qps['selectedJoinId'] = null;
            qps['selectedNodeId'] = null;

            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name,
                'run', this.currentNodeRun ? this.currentNodeRun.num : this._route.snapshot.params['number']], {
                    queryParams: Object.assign({}, qps, {
                        selectedNodeRunId: this.currentNodeRun ? this.currentNodeRun.id : -1,
                        selectedNodeRunNum: this.currentNodeRun ? this.currentNodeRun.num : 0,
                        selectedNodeId: this.node.id
                    })
                });
        }
    }

    displayDropdown(): void {
        this.elementRef.nativeElement.style.zIndex = 50;
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = 0;
    }

    openRunNode($event): void {
        $event.stopPropagation();
        this.workflowRunNode.show();
    }
}
