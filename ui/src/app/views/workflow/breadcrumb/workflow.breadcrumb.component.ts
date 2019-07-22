import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Project } from '../../../model/project.model';
import { Workflow } from '../../../model/workflow.model';
import { WorkflowRun } from '../../../model/workflow.run.model';
import { PathItem } from '../../../shared/breadcrumb/breadcrumb.component';

@Component({
    selector: 'app-workflow-breadcrumb',
    templateUrl: './workflow.breadcrumb.html',
    styleUrls: ['./workflow.breadcrumb.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowBreadCrumbComponent {
    _project: Project;
    @Input() set project(p: Project) {
        this._project = p;
        this.updatePath();
    }
    get project() { return this._project; }

    _workflow: Workflow;
    @Input() set workflow(w: Workflow) {
        this._workflow = w;
        this.updatePath();
    }
    get workflow() { return this._workflow; }

    _workflowRun: WorkflowRun;
    @Input() set workflowRun(wr: WorkflowRun) {
        this._workflowRun = wr;
        this.updatePath();
    }
    get workflowRun() { return this._workflowRun; }

    path: Array<PathItem>;

    updatePath() {
        let path = new Array<PathItem>();

        if (this._project) {
            path.push(<PathItem>{
                icon: 'browser',
                text: this._project.name,
                routerLink: ['/project', this._project.key],
                queryParams: { tab: 'workflows' }
            })

            if (this._workflow) {
                path.push(<PathItem>{
                    icon: 'share alternate',
                    text: this._workflow.name,
                    active: this._workflow && !this._workflowRun,
                    routerLink: ['/project', this._project.key, 'workflow', this._workflow.name],
                })

                if (this._workflowRun) {
                    path.push(<PathItem>{
                        icon: 'tag',
                        text: '' + this._workflowRun.num,
                        active: !!this._workflow.name && !!this._workflowRun.num,
                        routerLink: ['/project', this._project.key, 'workflow', this._workflow.name, 'run', this._workflowRun.num]
                    })
                }
            }
        }

        this.path = path;
    }
}
