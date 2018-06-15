import {AfterViewInit, Component, ElementRef, Input} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {cloneDeep} from 'lodash';
import {Project} from '../../../../model/project.model';
import {Workflow, WorkflowNode, WorkflowNodeHook, WorkflowNodeHookConfigValue} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
})
export class WorkflowNodeHookComponent implements AfterViewInit {

    _hook: WorkflowNodeHook;
    @Input('hook')
    set hook(data: WorkflowNodeHook) {
        if (data) {
            this._hook = data;
            if (this._hook.config['hookIcon']) {
                this.icon = (<WorkflowNodeHookConfigValue>this._hook.config['hookIcon']).value.toLowerCase();
            } else {
                this.icon = this._hook.model.icon.toLowerCase();
            }
        }
    }
    get hook() {
      return this._hook;
    }
    @Input() readonly = false;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() node: WorkflowNode;

    icon: string;
    loading = false;
    selectedHookId: number;

    constructor(private elementRef: ElementRef,
        private _route: ActivatedRoute,
        private _router: Router) {

        this._route.queryParams.subscribe((qp) => {
            if (qp['selectedHookId']) {
                this.selectedHookId = parseInt(qp['selectedHookId'], 10);
            } else {
                this.selectedHookId = null;
            }
        });
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = '5px';
    }

    openEditHookSidebar(): void {
        if (this.workflow.previewMode) {
          return;
        }

        let qps = cloneDeep(this._route.snapshot.queryParams);
        qps['selectedJoinId'] = null;
        qps['selectedNodeId'] = null;

        if (!this._route.snapshot.params['number']) {
            qps['selectedNodeRunId'] = null;
            qps['selectedNodeRunNum'] = null;
            qps['selectedJoinRunId'] = null;
            qps['selectedJoinRunNum'] = null;

            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name
            ], { queryParams: Object.assign({}, qps, {selectedHookId: this.hook.id })});
        } else {
            qps['selectedJoinId'] = null;
            qps['selectedNodeId'] = null;
            qps['selectedNodeRunId'] = null;
            qps['selectedNodeRunNum'] = null;
            qps['selectedJoinRunId'] = null;
            qps['selectedJoinRunNum'] = null;

            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name,
                'run', this._route.snapshot.params['number']], {
                    queryParams: Object.assign({}, qps, {
                        selectedHookId: this.hook.id
                    })
                });
        }
    }
}
