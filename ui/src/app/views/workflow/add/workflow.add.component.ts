import {Component} from '@angular/core';
import {Workflow} from '../../../model/workflow.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-add',
    templateUrl: './workflow.add.html',
    styleUrls: ['./workflow.add.scss']
})
export class WorkflowAddComponent {

    workflow: Workflow;
    project: Project;

    loading = false;


    constructor(private _activatedRoute: ActivatedRoute, private _router: Router, private _workflowStore: WorkflowStore,
                private _translate: TranslateService, private _toast: ToastService) {
        this.workflow = new Workflow();

        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });
    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], {queryParams: {tab: 'workflows'}});
    }

    createWorkflow(): void {
        this.loading = true;
        this._workflowStore.addWorkflow(this.project.key, this.workflow).first().subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_added'));
            this.loading = false;
        }, () => {
            this.loading = false;
        });
    }
}
