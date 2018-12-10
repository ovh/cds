import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { Group } from '../../../../model/group.model';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { GroupService } from '../../../../service/services.module';
import { WorkflowTemplateService } from '../../../../service/workflow-template/workflow-template.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Tab } from '../../../../shared/tabs/tabs.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-template-edit',
    templateUrl: './workflow-template.edit.html',
    styleUrls: ['./workflow-template.edit.scss']
})
export class WorkflowTemplateEditComponent implements OnInit {
    oldWorkflowTemplate: WorkflowTemplate;
    workflowTemplate: WorkflowTemplate;
    groups: Array<Group>;
    loading: boolean;
    path: Array<PathItem>;
    tabs: Array<Tab>;
    selectedTab: Tab;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _groupService: GroupService,
        private _route: ActivatedRoute,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router
    ) { }

    ngOnInit() {
        this.tabs = [<Tab>{
            translate: 'workflow_template',
            icon: 'paste',
            key: 'workflow_template',
            default: true
        }, <Tab>{
            translate: 'common_audit',
            icon: 'history',
            key: 'audits'
        }];

        this._route.params.subscribe(params => {
            const groupName = params['groupName'];
            const templateSlug = params['templateSlug'];
            this.getTemplate(groupName, templateSlug);
        });

        this.getGroups();
    }

    getGroups() {
        this.loading = true;
        this._groupService.getGroups()
            .pipe(finalize(() => this.loading = false))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    getTemplate(groupName: string, templateSlug: string) {
        this.loading = true;
        this._workflowTemplateService.getWorkflowTemplate(groupName, templateSlug)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wt => {
                this.oldWorkflowTemplate = { ...wt };
                this.workflowTemplate = wt;
                this.updatePath();
            });
    }

    saveWorkflowTemplate() {
        this.loading = true;
        this._workflowTemplateService.updateWorkflowTemplate(this.oldWorkflowTemplate, this.workflowTemplate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wt => {
                this.oldWorkflowTemplate = { ...wt };
                this.workflowTemplate = wt;
                this._toast.success('', this._translate.instant('workflow_template_saved'));
                this._router.navigate(['settings', 'workflow-template', this.workflowTemplate.group.name, this.workflowTemplate.slug]);
            });
    }

    deleteWorkflowTemplate() {
        this.loading = true;
        this._workflowTemplateService.deleteWorkflowTemplate(this.workflowTemplate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(_ => {
                this._toast.success('', this._translate.instant('workflow_template_deleted'));
                this._router.navigate(['settings', 'workflow-template']);
            });
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'workflow_templates',
            routerLink: ['/', 'settings', 'workflow-template']
        }];

        if (this.oldWorkflowTemplate) {
            this.path.push(<PathItem>{
                text: this.oldWorkflowTemplate.name,
                routerLink: ['/', 'settings', 'workflow-template', this.oldWorkflowTemplate.group.name, this.oldWorkflowTemplate.slug]
            });
        }
    }
}
