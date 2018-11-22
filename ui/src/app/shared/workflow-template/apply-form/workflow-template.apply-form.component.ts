import { Component, Input } from '@angular/core';
import { FormControl } from '@angular/forms';
import { Project } from '../../../model/project.model';
import { WorkflowTemplate } from '../../../model/workflow-template.model';

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

    parameterValues: any;

    loading: boolean;

    constructor() { }

    applyTemplate() {
        this._workflowTemplate.parameters.forEach(parameter => {
            if (parameter.type === 'boolean') {
                console.log(this.parameterValues[parameter.key].value);
            } else {
                console.log(this.parameterValues[parameter.key]);
            }
        });
    }
}
