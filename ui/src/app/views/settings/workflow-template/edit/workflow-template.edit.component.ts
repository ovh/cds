import {ChangeDetectorRef, Component, OnInit, ViewChild} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {AuditWorkflowTemplate} from 'app/model/audit.model';
import {Group} from 'app/model/group.model';
// tslint:disable-next-line: max-line-length
import {
    InstanceStatus,
    InstanceStatusUtil,
    WorkflowTemplate,
    WorkflowTemplateError,
    WorkflowTemplateInstance
} from 'app/model/workflow-template.model';
import {Workflow} from 'app/model/workflow.model';
import {GroupService} from 'app/service/group/group.service';
import {WorkflowTemplateService} from 'app/service/workflow-template/workflow-template.service';
import {PathItem} from 'app/shared/breadcrumb/breadcrumb.component';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {calculateWorkflowTemplateDiff} from 'app/shared/diff/diff';
import {Item} from 'app/shared/diff/list/diff.list.component';
import {Column, ColumnType} from 'app/shared/table/data-table.component';
import {Tab} from 'app/shared/tabs/tabs.component';
import {ToastService} from 'app/shared/toast/ToastService';
import {WorkflowTemplateApplyModalComponent} from 'app/shared/workflow-template/apply-modal/workflow-template.apply-modal.component';
import {WorkflowTemplateBulkModalComponent} from 'app/shared/workflow-template/bulk-modal/workflow-template.bulk-modal.component';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/internal/operators/finalize';
import {first} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-template-edit',
    templateUrl: './workflow-template.edit.html',
    styleUrls: ['./workflow-template.edit.scss']
})
@AutoUnsubscribe()
export class WorkflowTemplateEditComponent implements OnInit {

    @ViewChild('templateApplyModal')
    templateApplyModal: WorkflowTemplateApplyModalComponent;
    @ViewChild('templateBulkModal')
    templateBulkModal: WorkflowTemplateBulkModalComponent;

    oldWorkflowTemplate: WorkflowTemplate;
    workflowTemplate: WorkflowTemplate;
    groups: Array<Group>;
    audits: Array<AuditWorkflowTemplate>;
    instances: Array<WorkflowTemplateInstance>;
    usages: Array<Workflow>;
    loading: boolean;
    loadingInstances: boolean;
    loadingAudits: boolean;
    loadingUsage: boolean;
    path: Array<PathItem>;
    tabs: Array<Tab>;
    selectedTab: Tab;
    columnsAudits: Array<Column<AuditWorkflowTemplate>>;
    columnsInstances: Array<Column<WorkflowTemplateInstance>>;
    diffItems: Array<Item>;
    groupName: string;
    templateSlug: string;
    selectedWorkflowTemplateInstance: WorkflowTemplateInstance;
    errors: Array<WorkflowTemplateError>;
    paramsSub: Subscription;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _groupService: GroupService,
        private _route: ActivatedRoute,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) {
    }

    ngOnInit() {
        this.tabs = [<Tab>{
            translate: 'workflow_template',
            icon: 'paste',
            key: 'workflow_template',
            default: true
        }, <Tab>{
            translate: 'workflow_template_instances',
            icon: 'file outline',
            key: 'instances'
        }, <Tab>{
            translate: 'common_audit',
            icon: 'history',
            key: 'audits'
        }, <Tab>{
            translate: 'common_usage',
            icon: 'map signs',
            key: 'usage'
        }];

        this.columnsAudits = [
            <Column<AuditWorkflowTemplate>>{
                name: 'audit_modification_type',
                class: 'two',
                selector: (a: AuditWorkflowTemplate) => a.event_type
            },
            <Column<AuditWorkflowTemplate>>{
                name: 'common_version',
                class: 'one',
                selector: (a: AuditWorkflowTemplate) => a.data_after.version
            },
            <Column<AuditWorkflowTemplate>>{
                type: ColumnType.DATE,
                name: 'audit_time_author',
                class: 'two',
                selector: (a: AuditWorkflowTemplate) => a.created
            },
            <Column<AuditWorkflowTemplate>>{
                name: 'audit_username',
                class: 'two',
                selector: (a: AuditWorkflowTemplate) => a.triggered_by
            },
            <Column<AuditWorkflowTemplate>>{
                type: ColumnType.MARKDOWN,
                class: 'seven',
                name: 'common_description',
                selector: (a: AuditWorkflowTemplate) => a.change_message
            }
        ];

        this.columnsInstances = [
            <Column<WorkflowTemplateInstance>>{
                type: ColumnType.DATE,
                name: 'common_created',
                class: 'two',
                selector: (i: WorkflowTemplateInstance) => i.first_audit.created
            }, <Column<WorkflowTemplateInstance>>{
                name: 'common_created_by',
                class: 'two',
                selector: (i: WorkflowTemplateInstance) => i.first_audit.triggered_by
            }, <Column<WorkflowTemplateInstance>>{
                type: (i: WorkflowTemplateInstance) => {
                    let status = i.status(this.workflowTemplate);
                    return status === InstanceStatus.NOT_IMPORTED ? ColumnType.TEXT_LABELS : ColumnType.ROUTER_LINK_WITH_LABELS;
                },
                name: 'common_workflow',
                class: 'seven',
                selector: (i: WorkflowTemplateInstance) => {
                    let value = i.project.key + '/' + (i.workflow ? i.workflow.name : i.workflow_name);
                    let status = i.status(this.workflowTemplate);

                    let labels = [];
                    if (i.workflow && i.workflow.from_repository) {
                        labels.push({ color: 'blue', title: 'as code' });
                    }

                    return status === InstanceStatus.NOT_IMPORTED ? {
                        labels,
                        value
                    } : {
                        link: '/project/' + i.project.key + '/workflow/' + i.workflow.name,
                        labels,
                        value
                    };
                }
            }, <Column<WorkflowTemplateInstance>>{
                type: ColumnType.LABEL,
                name: 'common_status',
                class: 'three',
                selector: (i: WorkflowTemplateInstance) => {
                    let status = i.status(this.workflowTemplate);
                    return {
                        class: InstanceStatusUtil.color(status),
                        value: status
                    };
                }
            }, <Column<WorkflowTemplateInstance>>{
                type: ColumnType.BUTTON,
                name: 'common_action',
                class: 'two right aligned',
                selector: (i: WorkflowTemplateInstance) => {
                    return {
                        title: 'common_update',
                        class: 'primary small',
                        click: () => {
                            this.clickUpdate(i);
                        }
                    };
                }
            }
        ];

        this.paramsSub = this._route.params.subscribe(params => {
            this.groupName = params['groupName'];
            this.templateSlug = params['templateSlug'];
            this.getTemplate(this.groupName, this.templateSlug);
            this._cd.markForCheck();
        });

        this.getGroups();
    }

    getTemplate(groupName: string, templateSlug: string) {
        this.loading = true;
        this._workflowTemplateService.get(groupName, templateSlug)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wt => {
                this.oldWorkflowTemplate = {...wt};
                this.workflowTemplate = wt;

                if (this.workflowTemplate.editable) {
                    this.columnsAudits.push(<Column<AuditWorkflowTemplate>>{
                        type: ColumnType.CONFIRM_BUTTON,
                        name: 'common_action',
                        class: 'two right aligned',
                        selector: (a: AuditWorkflowTemplate) => {
                            return {
                                title: 'common_rollback',
                                click: () => {
                                    this.clickRollback(a)
                                }
                            };
                        }
                    });
                }

                this.updatePath();
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

    getAudits() {
        this.loadingAudits = true;
        this._workflowTemplateService.getAudits(this.groupName, this.templateSlug)
            .pipe(finalize(() => {
                this.loadingAudits = false;
                this._cd.markForCheck();
            }))
            .subscribe(as => {
                this.audits = as;
            });
    }

    saveWorkflowTemplate(wt: WorkflowTemplate) {
        this.loading = true;
        this._workflowTemplateService.update(this.oldWorkflowTemplate, wt)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                this.oldWorkflowTemplate = {...res};
                this.workflowTemplate = res;
                this.updatePath();
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
        this._workflowTemplateService.delete(this.workflowTemplate)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_template_deleted'));
                this._router.navigate(['settings', 'workflow-template']);
            });
    }

    selectTab(tab: Tab): void {
        switch (tab.key) {
            case 'instances':
                this.getInstances();
                break;
            case 'audits':
                this.getAudits();
                break;
            case 'usage':
                this.getUsage();
                break;
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

    clickAudit(a: AuditWorkflowTemplate): void {
        let before = a.data_before ? a.data_before : null;
        let after = a.data_after ? a.data_after : null;
        this.diffItems = calculateWorkflowTemplateDiff(before, after);
    }

    clickRollback(a: AuditWorkflowTemplate): void {
        let wt = a.data_before ? a.data_before : null;
        if (!wt) {
            wt = a.data_after ? a.data_after : null;
        }
        this.saveWorkflowTemplate(wt);
    }

    getUsage() {
        this.loadingUsage = true;
        this._workflowTemplateService.getUsage(this.groupName, this.templateSlug)
            .pipe(first())
            .pipe(finalize(() => {
                this.loadingUsage = false;
                this._cd.markForCheck();
            }))
            .subscribe((workflows) => this.usages = workflows);
    }

    getInstances() {
        this.loadingInstances = true;
        this._workflowTemplateService.getInstances(this.groupName, this.templateSlug)
            .pipe(finalize(() => {
                this.loadingInstances = false;
                this._cd.markForCheck();
            }))
            .subscribe(is => {
                this.instances = is.sort((a, b) => a.key() < b.key() ? -1 : 1)
                this.tabs = this.tabs.map((tab) => {
                    tab.default = false;
                    if (tab.key === 'instances') {
                        tab.translate = 'workflow_template_instances_with_nb';
                        tab.translate_args = {
                            nb: this.instances.length,
                        };
                    }

                    return tab;
                });
            });
    }

    clickCreateBulk() {
        this.templateBulkModal.show();
    }

    clickUpdate(i: WorkflowTemplateInstance) {
        this.selectedWorkflowTemplateInstance = i;
        this.templateApplyModal.show();
    }

    modalClose() {
        this.getInstances();
    }
}
