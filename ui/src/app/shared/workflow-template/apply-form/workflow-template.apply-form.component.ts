import { Component, Input } from '@angular/core';
import { FormControl } from '@angular/forms';
import { Router } from '@angular/router';
import { finalize, first } from 'rxjs/operators';
import { Project } from '../../../model/project.model';
import { WorkflowTemplate, WorkflowTemplateApplyResult, WorkflowTemplateRequest } from '../../../model/workflow-template.model';
import { RepoManagerService } from '../../../service/repomanager/project.repomanager.service';
import { WorkflowTemplateService } from '../../../service/workflow-template/workflow-template.service';

@Component({
    selector: 'app-workflow-template-apply-form',
    templateUrl: './workflow-template.apply-form.html',
    styleUrls: ['./workflow-template.apply-form.scss']
})
export class WorkflowTemplateApplyFormComponent {
    _workflowTemplate: WorkflowTemplate;

    @Input() project: Project;

    @Input() set workflowTemplate(wt: WorkflowTemplate) {
        this._workflowTemplate = wt;

        this.parameterValues = {};

        this._workflowTemplate.parameters.forEach(parameter => {
            if (parameter.type === 'boolean') {
                this.parameterValues[parameter.key] = new FormControl();
            }
        });
    }

    get workflowTemplate() { return this._workflowTemplate; }

    parameterName: string;
    parameterValues: any;

    loading: boolean;

    result: WorkflowTemplateApplyResult;

    constructor(private _repoManagerService: RepoManagerService,
        private _workflowTemplateService: WorkflowTemplateService,
        private _router: Router) { }

    applyTemplate() {
        let req = new WorkflowTemplateRequest();

        req.project_key = this.project.key;
        req.workflow_name = this.parameterName;
        req.parameters = {};

        this._workflowTemplate.parameters.forEach(parameter => {
            if (this.parameterValues[parameter.key]) {
                switch (parameter.type) {
                    case 'boolean':
                        req.parameters[parameter.key] = this.parameterValues[parameter.key] &&
                            !!this.parameterValues[parameter.key].value ? 'true' : 'false';
                        break;
                    case 'repository':
                        if (this.parameterValues[parameter.key + '-repository']) {
                            req.parameters[parameter.key] = this.parameterValues[parameter.key] + '/' +
                                this.parameterValues[parameter.key + '-repository'].fullname;
                        }
                        break;
                    default:
                        req.parameters[parameter.key] = this.parameterValues[parameter.key];
                        break;
                }
            }
        });

        this.result = null;
        this.loading = true;
        this._workflowTemplateService.applyWorkflowTemplate(this._workflowTemplate.group.name, this._workflowTemplate.slug, req)
            .pipe(first(), finalize(() => this.loading = false))
            .subscribe(res => {
                this.result = res;
            });
    }

    fetchRepos(parameterKey: string, repoMan: string): void {
        this._repoManagerService.getRepositories(this.project.key, repoMan, false).subscribe(rs => {
            this.parameterValues[parameterKey + '-repositories'] = rs;
        });
    }

    resyncRepos(parameterKey: string) {
        if (this.parameterValues[parameterKey]) {
            this.loading = true;
            this._repoManagerService.getRepositories(this.project.key, this.parameterValues[parameterKey], true)
                .pipe(first(), finalize(() => this.loading = false))
                .subscribe(rs => this.parameterValues[parameterKey + '-repositories'] = rs);
        }
    }

    goToWorkflow(): void {
        this._router.navigate(['/project', this.project.key, 'workflow', this.result.workflow_name]);
    }
}
