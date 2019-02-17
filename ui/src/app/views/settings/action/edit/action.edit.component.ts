import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Action, Usage } from '../../../../model/action.model';
import { AuditAction } from '../../../../model/audit.model';
import { Group } from '../../../../model/group.model';
import { ActionService } from '../../../../service/action/action.service';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
import { GroupService } from '../../../../service/group/group.service';
import { ActionEvent } from '../../../../shared/action/action.event.model';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { Item } from '../../../../shared/diff/list/diff.list.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';
import { Tab } from '../../../../shared/tabs/tabs.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-action-edit',
    templateUrl: './action.edit.html',
    styleUrls: ['./action.edit.scss']
})
@AutoUnsubscribe()
export class ActionEditComponent implements OnInit {
    action: Action;
    actionDoc: string;
    isAdmin: boolean;
    loadingUsage: boolean;
    usage: Usage;
    path: Array<PathItem>;
    paramsSub: Subscription;
    loading: boolean;
    groups: Array<Group>;
    tabs: Array<Tab>;
    selectedTab: Tab;
    groupName: string;
    actionName: string;
    audits: Array<AuditAction>;
    columnsAudits: Array<Column<AuditAction>>;
    diffItems: Array<Item>;

    constructor(
        private _actionService: ActionService,
        private _toast: ToastService, private _translate: TranslateService,
        private _route: ActivatedRoute, private _router: Router,
        private _authentificationStore: AuthentificationStore,
        private _groupService: GroupService
    ) {
        if (this._authentificationStore.isConnected()) {
            this.isAdmin = this._authentificationStore.isAdmin();
        }
    }

    ngOnInit() {
        this.tabs = [<Tab>{
            translate: 'common_action',
            icon: 'list',
            key: 'action',
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

        this.columnsAudits = [
            <Column<AuditAction>>{
                name: 'audit_modification_type',
                class: 'two',
                selector: (a: AuditAction) => a.event_type
            },
            <Column<AuditAction>>{
                type: ColumnType.DATE,
                name: 'audit_time_author',
                class: 'two',
                selector: (a: AuditAction) => a.created
            },
            <Column<AuditAction>>{
                name: 'audit_username',
                class: 'two',
                selector: (a: AuditAction) => a.triggered_by
            },
        ];

        this.paramsSub = this._route.params.subscribe(params => {
            this.groupName = params['groupName'];
            this.actionName = params['actionName'];
            this.getAction();
        });

        this.getGroups();
    }

    selectTab(tab: Tab): void {
        switch (tab.key) {
            case 'audits':
                this.getAudits();
                break;
            case 'usage':
                this.getUsage();
                break;
        }
        this.selectedTab = tab;
    }

    getAction(): void {
        this._actionService.get(this.groupName, this.actionName).subscribe(u => {
            this.action = u;
            let re = /\s/gi;
            this.actionDoc = u.name.toLowerCase().replace(re, '-');
            this.updatePath();
        });
    }

    getUsage(): void {
        this.loadingUsage = true;
        this._actionService.getUsage(this.groupName, this.actionName)
            .pipe(finalize(() => this.loadingUsage = false))
            .pipe(first()).subscribe(p => {
                this.usage = p;
            });
    }

    getAudits(): void {
        this._actionService.getAudits(this.groupName, this.actionName).pipe(first()).subscribe(as => {
            this.audits = as;
        });
    }

    clickAudit(a: AuditAction) {
        let before = a.data_before ? a.data_before : null;
        let after = a.data_after ? a.data_after : null;
        this.diffItems = [<Item>{
            before: before ? JSON.stringify(before) : null,
            after: after ? JSON.stringify(after) : null,
            type: 'application/json'
        }]
    }

    actionEvent(event: ActionEvent): void {
        event.action.loading = true;

        if (event.action.actions) {
            event.action.actions.forEach(a => {
                if (a.parameters) {
                    a.parameters.forEach(p => {
                        if (p.type === 'boolean' && !p.value) {
                            p.value = 'false';
                        }
                        p.value = p.value.toString();
                    });
                }
            });
        }
        if (event.action.parameters) {
            event.action.parameters.forEach(p => {
                if (p.type === 'boolean' && !p.value) {
                    p.value = 'false';
                }
                p.value = p.value.toString();
            });
        }

        switch (event.type) {
            case 'update':
                this._actionService.update(this.action, event.action).subscribe(action => {
                    this._toast.success('', this._translate.instant('action_saved'));
                    this.action = action;
                }, () => {
                    this.action.loading = false;
                });
                break;
            case 'delete':
                this._actionService.deleteAction(event.action.name).subscribe(() => {
                    this._toast.success('', this._translate.instant('action_deleted'));
                    this._router.navigate(['settings', 'action']);
                }, () => {
                    this.action.loading = false;
                });
                break;
        }
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'action_list_title',
            routerLink: ['/', 'settings', 'action']
        }];

        if (this.action && this.action.id) {
            this.path.push(<PathItem>{
                text: this.action.name + ' - ' + this.action.type,
                routerLink: ['/', 'settings', 'action', this.action.name]
            });
        }
    }

    getGroups() {
        this.loading = true;
        this._groupService.getGroups()
            .pipe(finalize(() => this.loading = false))
            .subscribe(gs => {
                this.groups = gs;
            });
    }
}
