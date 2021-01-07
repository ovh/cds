import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, Output } from '@angular/core';
import { Group } from 'app/model/group.model';
import {
    WorkflowTemplate,
    WorkflowTemplateError,
    WorkflowTemplateParameter
} from 'app/model/workflow-template.model';
import { Base64 } from 'app/shared/base64.utils';
import { SharedService } from 'app/shared/shared.service';

@Component({
    selector: 'app-workflow-template-form',
    templateUrl: './workflow-template.form.html',
    styleUrls: ['./workflow-template.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTemplateFormComponent {
    @Input() mode: string;
    @Input() groups: Array<Group>;
    @Input() loading: boolean;
    @Output() save = new EventEmitter();
    @Output() delete = new EventEmitter();

    _workflowTemplate: WorkflowTemplate;
    @Input() set workflowTemplate(wt: WorkflowTemplate) {
        this._workflowTemplate = { ...wt };

        if (!this._workflowTemplate) {
            this._workflowTemplate = <WorkflowTemplate>{ editable: true };
        }

        this.importFromURL = !!this._workflowTemplate.import_url;

        this.changeMessage = null;

        this.parameterKeys = [];
        this.parameterValues = {};
        if (this._workflowTemplate.parameters) {
            this._workflowTemplate.parameters.forEach((p, i) => {
                this.parameterValues[p.key] = p;
                this.parameterKeys.push(p.key);
            });
        }

        if (this._workflowTemplate.value) {
            this.workflowValue = Base64.b64DecodeUnicode(this._workflowTemplate.value);
        }

        this.pipelineValues = {};
        this.pipelineKeys = [];
        if (this._workflowTemplate.pipelines) {
            this._workflowTemplate.pipelines.forEach((p, i) => {
                this.pipelineValues[i] = Base64.b64DecodeUnicode(p.value);
                this.pipelineKeys.push(i);
            });
        }

        this.applicationValues = {};
        this.applicationKeys = [];
        if (this._workflowTemplate.applications) {
            this._workflowTemplate.applications.forEach((a, i) => {
                this.applicationValues[i] = Base64.b64DecodeUnicode(a.value);
                this.applicationKeys.push(i);
            });
        }

        this.environmentValues = {};
        this.environmentKeys = [];
        if (this._workflowTemplate.environments) {
            this._workflowTemplate.environments.forEach((e, i) => {
                this.environmentValues[i] = Base64.b64DecodeUnicode(e.value);
                this.environmentKeys.push(i);
            });
        }

        this.descriptionChange();
    }
    get workflowTemplate() {
 return this._workflowTemplate;
}

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
    changeMessage: string;
    importFromURL: boolean;

    constructor(
        private _sharedService: SharedService,
        private _cd: ChangeDetectorRef
    ) {
        this.templateParameterTypes = ['boolean', 'string', 'repository', 'json', 'ssh-key', 'pgp-key'];

        this.resetParameterValue();
    }

    resetParameterValue() {
        this.parameterValueAdd = new WorkflowTemplateParameter();
    }

    descriptionChange() {
        this.descriptionRows = this.getDescriptionHeight();
    }

    getDescriptionHeight(): number {
        return this._sharedService.getTextAreaheight(this.workflowTemplate.description);
    }

    clickSave() {
        if (this.importFromURL) {
            this.save.emit({ import_url: this.workflowTemplate.import_url })
            return;
        }

        if (!this.workflowTemplate.name || !this.workflowTemplate.group_id) {
            return;
        }

        this.save.emit({
            ...this.workflowTemplate,
            import_url: null,
            group_id: Number(this.workflowTemplate.group_id),
            value: this.workflowValue ? Base64.b64EncodeUnicode(this.workflowValue) : '',
            pipelines: Object.keys(this.pipelineValues).map(k => ({ value: this.pipelineValues[k] ? Base64.b64EncodeUnicode(this.pipelineValues[k]) : '' })),
            applications: Object.keys(this.applicationValues).map(k => ({ value: this.applicationValues[k] ? Base64.b64EncodeUnicode(this.applicationValues[k]) : '' })),
            environments: Object.keys(this.environmentValues).map(k => ({ value: this.environmentValues[k] ? Base64.b64EncodeUnicode(this.environmentValues[k]) : '' })),
            parameters: Object.keys(this.parameterValues).map(k => this.parameterValues[k]),
            change_message: this.changeMessage
        });
    }

    clickDelete() {
        this.delete.emit();
    }

    clickAddPipeline() {
        this.pipelineKeys.push(this.pipelineKeys.length);
    }

    clickRemovePipeline(key: number) {
        this.pipelineKeys = this.pipelineKeys.filter(k => k !== key);
        delete (this.pipelineValues[key]);
    }

    clickAddApplication() {
        this.applicationKeys.push(this.applicationKeys.length);
    }

    clickRemoveApplication(key: number) {
        this.applicationKeys = this.applicationKeys.filter(k => k !== key);
        delete (this.applicationValues[key]);
    }

    clickAddEnvironment() {
        this.environmentKeys.push(this.environmentKeys.length);
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

    changeFromURL() {
        this.importFromURL = !this.importFromURL;
        this._cd.markForCheck();
    }
}
