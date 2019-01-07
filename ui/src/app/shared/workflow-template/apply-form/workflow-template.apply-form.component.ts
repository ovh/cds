import { Component, EventEmitter, Input, OnChanges, Output } from '@angular/core';
import { Router } from '@angular/router';
import { finalize, first } from 'rxjs/operators';
import { Project } from '../../../model/project.model';
import {
    WorkflowTemplate,
    WorkflowTemplateApplyResult,
    WorkflowTemplateInstance,
    WorkflowTemplateRequest
} from '../../../model/workflow-template.model';
import { WorkflowTemplateService } from '../../../service/workflow-template/workflow-template.service';

@Component({
    selector: 'app-workflow-template-apply-form',
    templateUrl: './workflow-template.apply-form.html',
    styleUrls: ['./workflow-template.apply-form.scss']
})
export class WorkflowTemplateApplyFormComponent implements OnChanges {
    @Input() project: Project;
    @Input() workflowTemplate: WorkflowTemplate;
    @Input() workflowTemplateInstance: WorkflowTemplateInstance;
    @Input() withClose: boolean;
    @Output() close = new EventEmitter<number>();
    @Output() apply = new EventEmitter<number>();

    loading: boolean;
    result: WorkflowTemplateApplyResult;
    parameterName: string;
    parameterValues: { [key: string]: string; };

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _router: Router
    ) { }

    ngOnChanges() {
        this.parameterName = this.workflowTemplateInstance.request.workflow_name;
    }

    applyTemplate() {
        let req = new WorkflowTemplateRequest();

        req.project_key = this.project.key;
        req.workflow_name = this.parameterName;
        req.parameters = this.parameterValues;

        this.result = null;
        this.loading = true;
        this._workflowTemplateService.apply(this.workflowTemplate.group.name, this.workflowTemplate.slug, req)
            .pipe(first(), finalize(() => this.loading = false))
            .subscribe(res => {
                this.result = res;
                this.apply.emit();
            });
    }

    goToWorkflow(): void {
        this._router.navigate(['/project', this.project.key, 'workflow', this.result.workflow_name]);
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

    changeParam(values: { [key: string]: string; }) {
        this.parameterValues = values;
    }
}
