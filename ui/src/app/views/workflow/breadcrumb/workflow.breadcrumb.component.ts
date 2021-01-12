import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { Select } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowRun } from 'app/model/workflow.run.model';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-breadcrumb',
    templateUrl: './workflow.breadcrumb.html',
    styleUrls: ['./workflow.breadcrumb.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowBreadCrumbComponent implements OnInit, OnDestroy {
    _project: Project;
    @Input() set project(p: Project) {
        this._project = p;
        this.updatePath();
    }
    get project() {
 return this._project;
}

    _workflow: Workflow;
    @Input() set workflow(w: Workflow) {
        this._workflow = w;
        this.updatePath();
    }
    get workflow() {
 return this._workflow;
}

    @Select(WorkflowState.getSelectedWorkflowRun()) workflowRun$: Observable<WorkflowRun>;
    workflowRunSub: Subscription;
    workflowRun: WorkflowRun;

    path: Array<PathItem>;

    constructor(private _cd: ChangeDetectorRef) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.workflowRunSub = this.workflowRun$.subscribe(wr => {
            if (!wr && !this.workflowRun) {
                return;
            }
            if (wr && this.workflowRun && wr.id === this.workflowRun.id && wr.version === this.workflowRun.version) {
                return;
            }
            this.workflowRun = wr;
            this.updatePath();
            this._cd.detectChanges();
        });
    }

    updatePath() {
        let path = new Array<PathItem>();

        if (this._project) {
            path.push(<PathItem>{
                icon: 'browser',
                text: this._project.name,
                routerLink: ['/project', this._project.key],
                queryParams: { tab: 'workflows' }
            });

            if (this._workflow) {
                path.push(<PathItem>{
                    icon: 'share alternate',
                    text: this._workflow.name,
                    active: this._workflow && !this.workflowRun,
                    routerLink: ['/project', this._project.key, 'workflow', this._workflow.name],
                });

                if (this.workflowRun) {
                    path.push(<PathItem>{
                        icon: 'tag',
                        text: '' + (this.workflowRun.version ? this.workflowRun.version : this.workflowRun.num),
                        active: !!this._workflow.name && !!this.workflowRun.num,
                        routerLink: ['/project', this._project.key, 'workflow', this._workflow.name, 'run', this.workflowRun.num]
                    })
                }
            }
        }

        this.path = path;
    }
}
