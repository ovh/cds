import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { AuditWorkflowTemplate } from '../../../../model/audit.model';
import { Group } from '../../../../model/group.model';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { GroupService } from '../../../../service/services.module';
import { WorkflowTemplateService } from '../../../../service/workflow-template/workflow-template.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Item } from '../../../../shared/diff/list/diff.list.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';
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
    audits: Array<AuditWorkflowTemplate>;
    loading: boolean;
    path: Array<PathItem>;
    tabs: Array<Tab>;
    selectedTab: Tab;
    columns: Array<Column>;
    diffItems: Array<Item>;

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

        this.columns = [
            <Column>{
                name: this._translate.instant('audit_modification_type'),
                selector: a => a.event_type
            },
            <Column>{
                type: ColumnType.DATE,
                name: this._translate.instant('audit_time_author'),
                selector: a => a.created
            },
            <Column>{
                name: this._translate.instant('audit_username'),
                selector: a => a.triggered_by
            }
        ];

        this._route.params.subscribe(params => {
            const groupName = params['groupName'];
            const templateSlug = params['templateSlug'];
            this.getTemplate(groupName, templateSlug);
            this.getAudits(groupName, templateSlug);
        });

        this.getGroups();
    }

    getTemplate(groupName: string, templateSlug: string) {
        this.loading = true;
        this._workflowTemplateService.getWorkflowTemplate(groupName, templateSlug)
            .pipe(finalize(() => {
                this.loading = false
            }))
            .subscribe(wt => {
                this.oldWorkflowTemplate = { ...wt };
                this.workflowTemplate = wt;
                this.updatePath();
            });
    }

    getGroups() {
        this.loading = true;
        this._groupService.getGroups()
            .pipe(finalize(() => this.loading = false))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    getAudits(groupName: string, templateSlug: string) {
        this.loading = true;
        this._workflowTemplateService.getAudits(groupName, templateSlug)
            .pipe(finalize(() => this.loading = false))
            .subscribe(as => {
                this.audits = as.sort((a, b) => Date.parse(a.created) >= Date.parse(b.created) ? -1 : 1);
            });
    }

    saveWorkflowTemplate() {
        this.loading = true;
        this._workflowTemplateService.updateWorkflowTemplate(this.oldWorkflowTemplate, this.workflowTemplate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wt => {
                if (this.oldWorkflowTemplate.group.name === wt.group.name && this.oldWorkflowTemplate.slug === wt.slug) {
                    this.getAudits(wt.group.name, wt.slug);
                }
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

    clickAudit(a: AuditWorkflowTemplate) {
        let before = a.data_before ? <WorkflowTemplate>JSON.parse(a.data_before) : null;
        let after = a.data_after ? <WorkflowTemplate>JSON.parse(a.data_after) : null;

        let beforeTemplate: any;
        if (before) {
            beforeTemplate = {
                name: before.name,
                slug: before.slug,
                group_id: before.group_id,
                description: before.description,
                parameters: before.parameters
            }
        }

        let afterTemplate: any;
        if (after) {
            afterTemplate = {
                name: after.name,
                slug: after.slug,
                group_id: after.group_id,
                description: after.description,
                parameters: after.parameters
            }
        }

        let diffItems = [
            <Item>{
                name: 'template',
                before: beforeTemplate ? JSON.stringify(beforeTemplate) : null,
                after: afterTemplate ? JSON.stringify(afterTemplate) : null,
                type: 'application/json'
            },
            <Item>{
                name: 'workflow',
                before: before ? atob(before.value) : null,
                after: after ? atob(after.value) : null,
                type: 'text/x-yaml'
            }
        ];

        let pipelinesLength = Math.max(before ? before.pipelines.length : 0, after ? after.pipelines.length : 0);
        for (let i = 0; i < pipelinesLength; i++) {
            diffItems.push(
                <Item>{
                    name: 'pipeline ' + i,
                    before: before && before.pipelines[i] ? atob(before.pipelines[i].value) : null,
                    after: after && after.pipelines[i] ? atob(after.pipelines[i].value) : null,
                    type: 'text/x-yaml'
                })
        }

        let applicationsLength = Math.max(before ? before.applications.length : 0, after ? after.applications.length : 0);
        for (let i = 0; i < applicationsLength; i++) {
            diffItems.push(
                <Item>{
                    name: 'application ' + i,
                    before: before && before.applications[i] ? atob(before.applications[i].value) : null,
                    after: after && after.applications[i] ? atob(after.applications[i].value) : null,
                    type: 'text/x-yaml'
                })
        }

        let environmentsLength = Math.max(before ? before.environments.length : 0, after ? after.environments.length : 0);
        for (let i = 0; i < environmentsLength; i++) {
            diffItems.push(
                <Item>{
                    name: 'environment ' + i,
                    before: before && before.environments[i] ? atob(before.environments[i].value) : null,
                    after: after && after.environments[i] ? atob(after.environments[i].value) : null,
                    type: 'text/x-yaml'
                })
        }

        this.diffItems = diffItems;
    }
}
