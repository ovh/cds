import {Component, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode} from '../../../model/workflow.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {first, finalize} from 'rxjs/operators';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';

@Component({
    selector: 'app-workflow-add',
    templateUrl: './workflow.add.html',
    styleUrls: ['./workflow.add.scss']
})
export class WorkflowAddComponent {

    workflow: Workflow;
    project: Project;

    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    codeMirrorConfig: any;
    wfToImport: string;

    updated = false;
    loading = false;
    currentStep = 0;

    constructor(private _activatedRoute: ActivatedRoute,
                private _router: Router, private _workflowStore: WorkflowStore,
                private _translate: TranslateService, private _toast: ToastService) {
        this.workflow = new Workflow();

        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };
    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], {queryParams: {tab: 'workflows'}});
    }

    createWorkflow(node: WorkflowNode): void {
        this.loading = true;
        this.workflow.root = node;
        this._workflowStore.addWorkflow(this.project.key, this.workflow)
            .pipe(
                first(),
                finalize(() => this.loading = false)
            )
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_added'));
                this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name]);
            });
    }

    goToNextStep(stepNum: number): void {
      if (stepNum != null) {
        this.currentStep = stepNum;
      } else {
        this.currentStep++;
      }
    }

    importWorkflow() {
        this.loading = true;
        this._workflowStore.importWorkflow(this.project.key, this.workflow.name, this.wfToImport)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_added'));
                this.goToProject();
            });
    }
}
