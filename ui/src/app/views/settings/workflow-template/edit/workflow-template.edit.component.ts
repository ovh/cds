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
import { calculateWorkflowTemplateDiff } from '../../../../shared/diff/diff';
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
            },
            <Column>{
                type: ColumnType.MARKDOWN,
                name: this._translate.instant('common_description'),
                selector: a => a.change_message
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
                this.audits = as;
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
        this.diffItems = calculateWorkflowTemplateDiff(before, after);
    }
}
