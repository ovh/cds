import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Group } from '../../../../model/group.model';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { SharedService } from '../../../../shared/shared.service';

@Component({
    selector: 'app-workflow-template-form',
    templateUrl: './workflow-template.form.html',
    styleUrls: ['./workflow-template.form.scss']
})
export class WorkflowTemplateFormComponent {
    codeMirrorConfig: any;
    _workflowTemplate: WorkflowTemplate;
    workflowValue: string;
    pipelineValues: any;
    pipelineKeys: Array<number>;
    pipelineValueAdd: string;
    applicationValues: any;
    applicationKeys: Array<number>;
    applicationValueAdd: string;
    descriptionRows: number;
    parameterKeys: Array<string>;
    parameterValues: any;
    parameterValueAdd: any;
    templateParameterTypes: Array<string>;
    @Input() mode: string;
    @Input() groups: Array<Group>;
    @Input() loading: boolean;
    @Output() save = new EventEmitter<WorkflowTemplate>();
    @Output() delete = new EventEmitter<WorkflowTemplate>();

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
            wt.pipelines.map(p => atob(p.value)).forEach((p, i) => {
                this.pipelineValues[i] = p;
                this.pipelineKeys.push(i);
            });
        }

        this.applicationValues = {};
        this.applicationKeys = [];
        if (wt.applications) {
            wt.applications.map(a => atob(a.value)).forEach((a, i) => {
                this.applicationValues[i] = a;
                this.applicationKeys.push(i);
            });
        }

        this.descriptionChange();
    }
    get workflowTemplate() { return this._workflowTemplate; }

    constructor(
        private _sharedService: SharedService,
    ) {
        this.templateParameterTypes = ['boolean', 'string'];

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
        this._workflowTemplate.parameters = Object.keys(this.parameterValues).map(k => {
            return this.parameterValues[k];
        })

        if (!this._workflowTemplate.name || !this._workflowTemplate.group_id) {
            return;
        }

        if (this.workflowValue) {
            this._workflowTemplate.value = btoa(this.workflowValue);
        }
        this.workflowTemplate.group_id = Number(this.workflowTemplate.group_id);

        this.save.emit(this._workflowTemplate);
    }

    clickDelete() {
        this.delete.emit(this._workflowTemplate);
    }

    clickAddPipeline() {
        let k = this.pipelineKeys[this.pipelineKeys.length - 1] + 1;
        this.pipelineKeys.push(k)
        this.pipelineValues[k] = this.pipelineValueAdd;
        this.pipelineValueAdd = '';
    }

    clickRemovePipeline(key: number) {
        this.pipelineKeys = this.pipelineKeys.filter(k => k !== key);
        delete (this.pipelineValues[key]);
    }

    clickAddApplication() {
        let k = this.applicationKeys[this.applicationKeys.length - 1] + 1;
        this.applicationKeys.push(k)
        this.applicationValues[k] = this.applicationValueAdd;
        this.applicationValueAdd = '';
    }

    clickRemoveApplication(key: number) {
        this.applicationKeys = this.applicationKeys.filter(k => k !== key);
        delete (this.applicationValues[key]);
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
