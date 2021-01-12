import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { GroupService } from 'app/service/group/group.service';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Group } from '../../../../model/group.model';
import { PipelineTemplate, WorkflowTemplate, WorkflowTemplateError } from '../../../../model/workflow-template.model';
import { WorkflowTemplateService } from '../../../../service/workflow-template/workflow-template.service';
import { WorkflowService } from '../../../../service/workflow/workflow.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-template-add',
    templateUrl: './workflow-template.add.html',
    styleUrls: ['./workflow-template.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowTemplateAddComponent implements OnInit, OnDestroy {
    workflowTemplate: WorkflowTemplate;
    groups: Array<Group>;
    loading: boolean;
    path: Array<PathItem>;
    errors: Array<WorkflowTemplateError>;
    queryParamsSub: Subscription;
    projectKey: string;
    workflowName: string;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _workflowService: WorkflowService,
        private _groupService: GroupService,
        private _router: Router,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef
    ) {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'workflow_templates',
            routerLink: ['/', 'settings', 'workflow-template']
        }, <PathItem>{
            translate: 'common_create'
        }];

        this.workflowTemplate = <WorkflowTemplate>{ editable: true };
        this.getGroups();
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['from']) {
                let path = params['from'].split('/');
                if (path.length === 2) {
                    this.initFromWorkflow(path[0], path[1]);
                }
                this._cd.markForCheck();
            }
        });
    }

    initFromWorkflow(projectKey: string, workflowName: string) {
        this._workflowService.pullWorkflow(projectKey, workflowName)
            .subscribe(w => {
            this.projectKey = projectKey;
            this.workflowName = workflowName;
            let wt = <WorkflowTemplate>{
                editable: true,
                value: w.workflow
            };
            if (w.pipelines) {
                wt.pipelines = w.pipelines.map(p => <PipelineTemplate>{ value: p });
            }
            if (w.applications) {
                wt.applications = w.applications.map(a => <PipelineTemplate>{ value: a });
            }
            if (w.environments) {
                wt.environments = w.environments.map(e => <PipelineTemplate>{ value: e });
            }
            this.workflowTemplate = wt;
            this._cd.markForCheck();
        });
    }

    getGroups() {
        this.loading = true;
        this._groupService.getAll()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    saveWorkflowTemplate(wt: WorkflowTemplate) {
        this.loading = true;
        this._workflowTemplateService.add(wt)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                this.workflowTemplate = res;
                this.errors = [];
                this._toast.success('', this._translate.instant('workflow_template_created'));
                this._router.navigate(['settings', 'workflow-template', this.workflowTemplate.group.name, this.workflowTemplate.slug]);
            }, e => {
                if (e.error) {
                    this.errors = e.error.data;
                }
            });
    }
}
