import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Group } from '../../../../model/group.model';
import { User } from '../../../../model/user.model';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
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

    @Input() set workflowTemplate(wt: WorkflowTemplate) {
        if (!wt) {
            wt = new WorkflowTemplate();
        }

        this._workflowTemplate = wt;

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

        this.codeMirrorConfig.readOnly = !this._workflowTemplate.editable;
    }
    get workflowTemplate() { return this._workflowTemplate; }

    codeMirrorConfig: any;

    _workflowTemplate: WorkflowTemplate;
    descriptionRows: number;

    templateParameterTypes: Array<string>;
    parameterKeys: Array<string>;
    parameterValues: any;
    parameterValueAdd: any;

    workflowValue: string;

    pipelineValues: any;
    pipelineKeys: Array<number>;
    pipelineValueAdd: string;

    applicationValues: any;
    applicationKeys: Array<number>;
    applicationValueAdd: string;

    environmentValues: any;
    environmentKeys: Array<number>;
    environmentValueAdd: string;

    user: User;

    constructor(private _sharedService: SharedService) {
        this.templateParameterTypes = ['boolean', 'string', 'repository'];

        this.resetParameterValue();

        this.codeMirrorConfig = this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'text/x-yaml',
            lineWrapping: true,
            autoRefresh: true,
            lineNumbers: true,
        };
    }

    resetParameterValue() {
        this.parameterValueAdd = {};
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

        this.save.emit();
    }

    clickDelete() {
        this.delete.emit();
    }

    clickAddPipeline() {
        let k = this.pipelineKeys.length;
        this.pipelineKeys.push(k)
        this.pipelineValues[k] = this.pipelineValueAdd;
        this.pipelineValueAdd = '';
    }

    clickRemovePipeline(key: number) {
        this.pipelineKeys = this.pipelineKeys.filter(k => k !== key);
        delete (this.pipelineValues[key]);
    }

    clickAddApplication() {
        let k = this.applicationKeys.length;
        this.applicationKeys.push(k)
        this.applicationValues[k] = this.applicationValueAdd;
        this.applicationValueAdd = '';
    }

    clickRemoveApplication(key: number) {
        this.applicationKeys = this.applicationKeys.filter(k => k !== key);
        delete (this.applicationValues[key]);
    }

    clickAddEnvironment() {
        let k = this.environmentKeys.length;
        this.environmentKeys.push(k)
        this.environmentValues[k] = this.environmentValueAdd;
        this.environmentValueAdd = '';
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
}
