import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Group } from '../../../../model/group.model';
import { User } from '../../../../model/user.model';
import { WorkflowTemplate, WorkflowTemplateError, WorkflowTemplateParameter } from '../../../../model/workflow-template.model';
import { SharedService } from '../../../../shared/shared.service';

@Component({
    selector: 'app-workflow-template-form',
    templateUrl: './workflow-template.form.html',
    styleUrls: ['./workflow-template.form.scss']
})
export class WorkflowTemplateFormComponent {
    @Input() mode: string;
    @Input() groups: Array<Group>;
    @Input() loading: boolean;
    @Output() save = new EventEmitter();
    @Output() delete = new EventEmitter();

    _workflowTemplate: WorkflowTemplate;
    @Input() set workflowTemplate(wt: WorkflowTemplate) {
        if (!wt) {
            wt = new WorkflowTemplate();
            wt.editable = true;
        }

        this._workflowTemplate = wt;
        this.changeMessage = null;

        this.parameterKeys = [];
        this.parameterValues = {};
        if (wt.parameters) {
            wt.parameters.forEach((p, i) => {
                this.parameterValues[p.key] = p;
                this.parameterKeys.push(p.key);
            });
        }

        if (wt.value) {
            this.workflowValue = atob(wt.value);
        }

        this.pipelineValues = {};
        this.pipelineKeys = [];
        if (wt.pipelines) {
            wt.pipelines.forEach((p, i) => {
                this.pipelineValues[i] = atob(p.value);
                this.pipelineKeys.push(i);
            });
        }

        this.applicationValues = {};
        this.applicationKeys = [];
        if (wt.applications) {
            wt.applications.forEach((a, i) => {
                this.applicationValues[i] = atob(a.value);
                this.applicationKeys.push(i);
            });
        }

        this.environmentValues = {};
        this.environmentKeys = [];
        if (wt.environments) {
            wt.environments.forEach((e, i) => {
                this.environmentValues[i] = atob(e.value);
                this.environmentKeys.push(i);
            });
        }

        this.descriptionChange();
    }
    get workflowTemplate() { return this._workflowTemplate; }

    @Input() set errors(es: Array<WorkflowTemplateError>) {
        this.workflowError = null;
        this.pipelineErrors = {};
        this.applicationErrors = {};
        this.environmentErrors = {};

        if (es) {
            es.forEach(e => {
                switch (e.type) {
                    case 'workflow':
                        this.workflowError = e;
                        break;
                    case 'pipeline':
                        this.pipelineErrors[e.number] = e;
                        break;
                    case 'application':
                        this.applicationErrors[e.number] = e;
                        break;
                    case 'environment':
                        this.environmentErrors[e.number] = e;
                        break;
                    default:
                        break;
                }
            });
        }
    }

    codeMirrorConfig: any;
    descriptionRows: number;
    templateParameterTypes: Array<string>;
    parameterKeys: Array<string>;

    parameterValues: { [key: number]: WorkflowTemplateParameter };
    parameterValueAdd: WorkflowTemplateParameter;
    workflowValue: string;
    workflowError: WorkflowTemplateError;
    pipelineValues: { [key: number]: string; };
    pipelineErrors: { [key: number]: WorkflowTemplateError; };
    pipelineKeys: Array<number>;
    applicationValues: { [key: number]: string; };
    applicationErrors: { [key: number]: WorkflowTemplateError; };
    applicationKeys: Array<number>;
    environmentValues: { [key: number]: string; };
    environmentErrors: { [key: number]: WorkflowTemplateError; };
    environmentKeys: Array<number>;
    user: User;
    changeMessage: string;

    constructor(
        private _sharedService: SharedService
    ) {
        this.templateParameterTypes = ['boolean', 'string', 'repository', 'json'];

        this.resetParameterValue();
    }

    resetParameterValue() {
        this.parameterValueAdd = new WorkflowTemplateParameter();
    }

    descriptionChange() {
        this.descriptionRows = this.getDescriptionHeight();
    }

    getDescriptionHeight(): number {
        return this._sharedService.getTextAreaheight(this._workflowTemplate.description);
    }

    clickSave() {
        this._workflowTemplate.pipelines = Object.keys(this.pipelineValues).map(k => {
            return { value: this.pipelineValues[k] ? btoa(this.pipelineValues[k]) : '' };
        });
        this._workflowTemplate.applications = Object.keys(this.applicationValues).map(k => {
            return { value: this.applicationValues[k] ? btoa(this.applicationValues[k]) : '' };
        });
        this._workflowTemplate.environments = Object.keys(this.environmentValues).map(k => {
            return { value: this.environmentValues[k] ? btoa(this.environmentValues[k]) : '' };
        });
        this._workflowTemplate.parameters = Object.keys(this.parameterValues).map(k => {
            return this.parameterValues[k];
        });

        if (!this._workflowTemplate.name || !this._workflowTemplate.group_id) {
            return;
        }

        if (this.workflowValue) {
            this._workflowTemplate.value = btoa(this.workflowValue);
        }
        this.workflowTemplate.group_id = Number(this.workflowTemplate.group_id);

        if (this.changeMessage) {
            this.workflowTemplate.change_message = this.changeMessage;
        }

        this.save.emit();
    }

    clickDelete() {
        this.delete.emit();
    }

    clickAddPipeline() {
        let k = this.pipelineKeys.length;
        this.pipelineKeys.push(k);
    }

    clickRemovePipeline(key: number) {
        this.pipelineKeys = this.pipelineKeys.filter(k => k !== key);
        delete (this.pipelineValues[key]);
    }

    clickAddApplication() {
        let k = this.applicationKeys.length;
        this.applicationKeys.push(k);
    }

    clickRemoveApplication(key: number) {
        this.applicationKeys = this.applicationKeys.filter(k => k !== key);
        delete (this.applicationValues[key]);
    }

    clickAddEnvironment() {
        let k = this.environmentKeys.length;
        this.environmentKeys.push(k);
    }

    clickRemoveEnvironment(key: number) {
        this.environmentKeys = this.environmentKeys.filter(k => k !== key);
        delete (this.environmentValues[key]);
    }

    clickAddParameter() {
        let k = this.parameterValueAdd.key;
        this.parameterKeys.push(k)
        this.parameterValues[k] = this.parameterValueAdd;
        this.resetParameterValue();
    }

    clickRemoveParameter(key: string) {
        this.parameterKeys = this.parameterKeys.filter(k => k !== key);
        delete (this.parameterValues[key]);
    }

    workflowValueChange(value: string) {
        this.workflowValue = value;
    }

    pipelineValueChange(key: number, value: string) {
        this.pipelineValues[key] = value;
    }

    applicationValueChange(key: number, value: string) {
        this.applicationValues[key] = value;
    }

    environmentValueChange(key: number, value: string) {
        this.environmentValues[key] = value;
    }
}
