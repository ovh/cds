import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormControl } from '@angular/forms';
import { Router } from '@angular/router';
import { finalize, first } from 'rxjs/operators';
import { Project } from '../../../model/project.model';
import {
    WorkflowTemplate,
    WorkflowTemplateApplyResult,
    WorkflowTemplateInstance,
    WorkflowTemplateRequest
} from '../../../model/workflow-template.model';
import { Workflow } from '../../../model/workflow.model';
import { RepoManagerService } from '../../../service/repomanager/project.repomanager.service';
import { WorkflowTemplateService } from '../../../service/workflow-template/workflow-template.service';

@Component({
    selector: 'app-workflow-template-apply-form',
    templateUrl: './workflow-template.apply-form.html',
    styleUrls: ['./workflow-template.apply-form.scss']
})
export class WorkflowTemplateApplyFormComponent {
    _project: Project;
    @Input() set project(p: Project) {
        this._project = p;

        if (this._project.vcs_servers) {
            this.vcsNames = this._project.vcs_servers.map(vcs => vcs.name);
        }
    }
    get project() { return this._project; }

    @Input() workflow: Workflow;

    _workflowTemplate: WorkflowTemplate;
    @Input() set workflowTemplate(wt: WorkflowTemplate) {
        this._workflowTemplate = wt;

        this.parameterValues = {};

        if (this._workflowTemplate.parameters) {
            this._workflowTemplate.parameters.forEach(parameter => {
                if (parameter.type === 'boolean') {
                    this.parameterValues[parameter.key] = new FormControl();
                }
            });
        }

        this.fillFormWithInstanceData();
    }
    get workflowTemplate() { return this._workflowTemplate; }

    _workflowTemplateInstance: WorkflowTemplateInstance;
    @Input() set workflowTemplateInstance(wti: WorkflowTemplateInstance) {
        this._workflowTemplateInstance = wti;
        this.fillFormWithInstanceData();
    }
    get workflowTemplateInstance() { return this._workflowTemplateInstance; }

    @Input() withClose: boolean;
    @Output() close = new EventEmitter<number>();

    @Output() apply = new EventEmitter<number>();

    vcsNames: Array<string>;
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

        if (this._workflowTemplate.parameters) {
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
                                    this.parameterValues[parameter.key + '-repository'];
                            }
                            break;
                        default:
                            req.parameters[parameter.key] = this.parameterValues[parameter.key];
                            break;
                    }
                }
            });
        }

        this.result = null;
        this.loading = true;
        this._workflowTemplateService.applyWorkflowTemplate(this._workflowTemplate.group.name, this._workflowTemplate.slug, req)
            .pipe(first(), finalize(() => this.loading = false))
            .subscribe(res => {
                // if the workflow name changed move to new workflow page
                this.result = res;

                // specific check for case where workflow name change in template
                if (res.workflow_name !== this.workflow.name) {
                    this._router.navigate(['/project', this.project.key, 'workflow', res.workflow_name]);
                } else {
                    this.apply.emit();
                }
            });
    }

    fetchRepos(parameterKey: string, repoMan: string): void {
        this._repoManagerService.getRepositories(this.project.key, repoMan, false).subscribe(rs => {
            let repoNames = rs.map(r => r.fullname);

            this.parameterValues[parameterKey + '-repositories'] = repoNames;

            if (this._workflowTemplateInstance && this._workflowTemplateInstance.request.parameters[parameterKey]) {
                let v = this._workflowTemplateInstance.request.parameters[parameterKey];
                let s = v.split('/');
                if (s.length > 1) {
                    let selectedRepo = s.splice(1, s.length - 1).join('/');
                    let existingRepo = repoNames.find(n => n === selectedRepo);
                    if (existingRepo) {
                        this.parameterValues[parameterKey + '-repository'] = existingRepo;
                    }
                }
            }
        });
    }

    resyncRepos(parameterKey: string) {
        if (this.parameterValues[parameterKey]) {
            this.loading = true;
            this._repoManagerService.getRepositories(this.project.key, this.parameterValues[parameterKey], true)
                .pipe(first(), finalize(() => this.loading = false))
                .subscribe(rs => {
                    this.parameterValues[parameterKey + '-repositories'] = rs.map(r => r.fullname);
                });
        }
    }

    goToWorkflow(): void {
        this._router.navigate(['/project', this.project.key, 'workflow', this.result.workflow_name]);
    }

    fillFormWithInstanceData(): void {
        if (this._workflowTemplate && this._workflowTemplateInstance) {
            this.parameterName = this._workflowTemplateInstance.request.workflow_name;
            if (this._workflowTemplate.parameters) {
                this._workflowTemplate.parameters.forEach(parameter => {

                    let v = this._workflowTemplateInstance.request.parameters[parameter.key];
                    if (v) {
                        switch (parameter.type) {
                            case 'boolean':
                                this.parameterValues[parameter.key].setValue(v === 'true');
                                break;
                            case 'repository':
                                let s = v.split('/');
                                if (s.length > 1) {
                                    let existingVcs = this.vcsNames.find(vcs => vcs === s[0]);
                                    if (existingVcs) {
                                        this.parameterValues[parameter.key] = existingVcs;
                                        this.fetchRepos(parameter.key, existingVcs);
                                    }
                                }
                                break;
                            default:
                                this.parameterValues[parameter.key] = v;
                                break;
                        }
                    }
                });
            }
        }
    }

    filterRepo(options: Array<string>, query: string): Array<string> | false {
        if (!options) {
            return false;
        }
        if (!query || query.length < 3) {
            return options.slice(0, 100);
        }
        let queryLowerCase = query.toLowerCase();
        return options.filter(name => name.toLowerCase().indexOf(queryLowerCase) !== -1);
    }

    clickClose() {
        this.close.emit();
    }
}
