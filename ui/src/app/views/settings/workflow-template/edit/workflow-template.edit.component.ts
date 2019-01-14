import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { first } from 'rxjs/operators';
import { AuditWorkflowTemplate } from '../../../../model/audit.model';
import { Group } from '../../../../model/group.model';
import { WorkflowTemplate, WorkflowTemplateError } from '../../../../model/workflow-template.model';
import { Workflow } from '../../../../model/workflow.model';
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
    workflowsLinked: Array<Workflow>;
    loading: boolean;
    loadingAudits: boolean;
    loadingUsage: boolean;
    path: Array<PathItem>;
    tabs: Array<Tab>;
    selectedTab: Tab;
    columns: Array<Column>;
    diffItems: Array<Item>;
    groupName: string;
    templateSlug: string;
    errors: Array<WorkflowTemplateError>;

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
        }, <Tab>{
            translate: 'common_usage',
            icon: 'map signs',
            key: 'usage'
        }];

        this.columns = [
            <Column>{
                name: 'audit_modification_type',
                class: 'two',
                selector: a => a.event_type
            },
            <Column>{
                type: ColumnType.DATE,
                class: 'two',
                name: 'audit_time_author',
                selector: a => a.created
            },
            <Column>{
                name: 'audit_username',
                class: 'two',
                selector: a => a.triggered_by
            },
            <Column>{
                type: ColumnType.MARKDOWN,
                class: 'eight',
                name: 'common_description',
                selector: a => a.change_message
            }
        ];

        this._route.params.subscribe(params => {
            this.groupName = params['groupName'];
            this.templateSlug = params['templateSlug'];
            this.getTemplate(this.groupName, this.templateSlug);
            this.getAudits(this.groupName, this.templateSlug);
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

                if (this.workflowTemplate.editable) {
                    this.columns.push(<Column>{
                        type: ColumnType.CONFIRM_BUTTON,
                        name: 'common_action',
                        class: 'two right aligned',
                        selector: a => {
                            return {
                                title: 'common_rollback',
                                click: () => { this.clickRollback(a) }
                            };
                        }
                    });
                }

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
        this.loadingAudits = true;
        this._workflowTemplateService.getAudits(groupName, templateSlug)
            .pipe(finalize(() => this.loadingAudits = false))
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
                this.errors = [];
                this._toast.success('', this._translate.instant('workflow_template_saved'));
                this._router.navigate(['settings', 'workflow-template', this.workflowTemplate.group.name, this.workflowTemplate.slug]);
            }, e => {
                if (e.error) {
                    this.errors = e.error.data;
                }
            });
    }

    deleteWorkflowTemplate() {
        this.loading = true;
        this._workflowTemplateService.deleteWorkflowTemplate(this.workflowTemplate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_template_deleted'));
                this._router.navigate(['settings', 'workflow-template']);
            });
    }

    selectTab(tab: Tab): void {
        if (tab.key === 'usage') {
            this.getUsage();
        }
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

    clickRollback(a: AuditWorkflowTemplate) {
        this.workflowTemplate = a.data_before ? <WorkflowTemplate>JSON.parse(a.data_before) : null;
        if (!this.workflowTemplate) {
            this.workflowTemplate = a.data_after ? <WorkflowTemplate>JSON.parse(a.data_after) : null;
        }
        this.saveWorkflowTemplate();
    }

    getUsage() {
        if (this.workflowsLinked) {
            return;
        }
        this.loadingUsage = true;
        this._workflowTemplateService.getWorkflowTemplateUsage(this.groupName, this.templateSlug)
            .pipe(first())
            .pipe(finalize(() => this.loadingUsage = false))
            .subscribe((workflows) => this.workflowsLinked = workflows);
    }
}
