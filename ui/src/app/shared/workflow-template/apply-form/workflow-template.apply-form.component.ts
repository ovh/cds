import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    Output
} from '@angular/core';
import { Router } from '@angular/router';
import { Project } from 'app/model/project.model';
import {
    ParamData,
    WorkflowTemplate,
    WorkflowTemplateApplyResult,
    WorkflowTemplateInstance,
    WorkflowTemplateRequest
} from 'app/model/workflow-template.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowTemplateService } from 'app/service/workflow-template/workflow-template.service';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-template-apply-form',
    templateUrl: './workflow-template.apply-form.html',
    styleUrls: ['./workflow-template.apply-form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTemplateApplyFormComponent implements OnChanges {
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() workflowTemplate: WorkflowTemplate;
    @Input() workflowTemplateInstance: WorkflowTemplateInstance;
    @Input() withClose: boolean;
    @Output() close = new EventEmitter<number>();
    @Output() apply = new EventEmitter<number>();

    loading: boolean;
    result: WorkflowTemplateApplyResult;
    parameterName: string;
    parameterValues: ParamData;
    detached: boolean;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _router: Router, private _cd: ChangeDetectorRef
    ) { }

    ngOnChanges() {
        this.parameterName = this.workflowTemplateInstance ? this.workflowTemplateInstance.request.workflow_name : '';
    }

    applyTemplate() {
        let req = <WorkflowTemplateRequest>{
            project_key: this.project.key,
            workflow_name: this.parameterName,
            parameters: this.parameterValues,
            detached: !!this.detached
        };

        this.result = null;
        this.loading = true;
        this._workflowTemplateService.apply(this.workflowTemplate.group.name, this.workflowTemplate.slug, req)
            .pipe(first(), finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                // if the workflow name changed move to new workflow page
                this.result = res;

                // specific check for case where workflow name change in template
                if (this.workflow && res.workflow_name !== this.workflow.name) {
                    this._router.navigate(['/project', this.project.key, 'workflow', res.workflow_name]);
                } else {
                    this.apply.emit();
                }
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

    clickDetach() {
        this._workflowTemplateService.deleteInstance(this.workflowTemplate, this.workflowTemplateInstance)
            .subscribe(() => {
            this.clickClose();
        });
    }

    onSelectDetachChange(e: any) {
        this.detached = !this.detached;
    }
}
