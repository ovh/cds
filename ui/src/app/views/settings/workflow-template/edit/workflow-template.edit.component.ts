import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { finalize } from 'rxjs/internal/operators/finalize';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { WorkflowTemplateService } from '../../../../service/workflow-template/workflow-template.service';
import { SharedService } from '../../../../shared/shared.service';

@Component({
    selector: 'app-workflow-template-edit',
    templateUrl: './workflow-template.edit.html',
    styleUrls: ['./workflow-template.edit.scss']
})
export class WorkflowTemplateEditComponent {
    codeMirrorConfig: any;
    workflowTemplate: WorkflowTemplate;
    loading: boolean;
    workflowValue: string;
    pipelineValues: any;
    pipelineKeys: Array<number>;
    pipelineValueAdd: string;
    descriptionRows: number;

    constructor(
        private _sharedService: SharedService,
        private _workflowTemplateService: WorkflowTemplateService,
        private _route: ActivatedRoute
    ) {
        this.initPipelineValue();

        this.codeMirrorConfig = this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'text/x-yaml',
            lineWrapping: true,
            autoRefresh: true,
            lineNumbers: true,
        };

        this._route.params.subscribe(params => {
            const id = params['id'];
            this.getTemplate(id);
        });
    }

    initPipelineValue() {
        this.pipelineValueAdd = '\n\n';
    }

    getTemplate(id: number) {
        this.loading = true;
        this._workflowTemplateService.getWorkflowTemplate(id)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wt => {
                this.setTemplate(wt);
            });
    }

    descriptionChange() {
        this.descriptionRows = this.getDescriptionHeight();
    }

    getDescriptionHeight(): number {
        return this._sharedService.getTextAreaheight(this.workflowTemplate.description);
    }

    clickSave() {
        this.workflowTemplate.value = btoa(this.workflowValue);
        this.workflowTemplate.pipelines = Object.keys(this.pipelineValues).map(k => {
            return { value: btoa(this.pipelineValues[k]) };
        });
        this.loading = true;
        this._workflowTemplateService.updateWorkflowTemplate(this.workflowTemplate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wt => {
                this.setTemplate(wt);
            });
    }

    clickAddPipeline() {
        let k = this.pipelineKeys[this.pipelineKeys.length - 1] + 1;
        this.pipelineKeys.push(k)
        this.pipelineValues[k] = this.pipelineValueAdd;
        this.initPipelineValue();
    }

    clickRemovePipeline(key: number) {
        this.pipelineKeys = this.pipelineKeys.filter(k => k !== key);
        delete (this.pipelineValues[key]);
    }

    setTemplate(wt: WorkflowTemplate) {
        this.workflowTemplate = wt;
        this.workflowValue = atob(wt.value);
        this.pipelineValues = {};
        this.pipelineKeys = [];
        wt.pipelines.map(p => atob(p.value)).forEach((p, i) => {
            this.pipelineValues[i] = p;
            this.pipelineKeys.push(i);
        });
        this.descriptionChange();
    }
}
