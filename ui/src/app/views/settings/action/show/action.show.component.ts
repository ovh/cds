import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Action, Usage } from '../../../../model/action.model';
import { ActionService } from '../../../../service/action/action.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { Tab } from '../../../../shared/tabs/tabs.component';

@Component({
    selector: 'app-action-show',
    templateUrl: './action.show.html',
    styleUrls: ['./action.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ActionShowComponent implements OnInit, OnDestroy {
    action: Action;
    actionDoc: string;
    loadingUsage: boolean;
    usage: Usage;
    path: Array<PathItem>;
    paramsSub: Subscription;
    loading: boolean;
    tabs: Array<Tab>;
    selectedTab: Tab;
    actionName: string;

    constructor(
        private _route: ActivatedRoute,
        private _actionService: ActionService,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.tabs = [<Tab>{
            translate: 'common_action',
            icon: 'list',
            key: 'action',
            default: true
        }, <Tab>{
            translate: 'common_usage',
            icon: 'map signs',
            key: 'usage'
        }];

        this.paramsSub = this._route.params.subscribe(params => {
            let actionName = params['actionName'];

            if (actionName !== this.actionName) {
                this.actionName = params['actionName'];

                this.getAction();

                if (this.selectedTab) {
                    this.selectTab(this.selectedTab);
                }
            }
            this._cd.markForCheck();
        });
    }

    selectTab(tab: Tab): void {
        switch (tab.key) {
            case 'usage':
                this.getUsage();
                break;
        }
        this.selectedTab = tab;
    }

    getAction(): void {
        this.loading = true;
        this._actionService.getBuiltin(this.actionName)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .pipe(first()).subscribe(a => {
                this.action = a;
                this.updatePath();
            });
    }

    getUsage(): void {
        this.loadingUsage = true;
        this._actionService.getBuiltinUsage(this.actionName)
            .pipe(finalize(() => {
                this.loadingUsage = false;
                this._cd.markForCheck();
            }))
            .pipe(first()).subscribe(u => {
                this.usage = u;
            });
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
                text: this.action.name,
                routerLink: ['/', 'settings', 'action-builtin', this.action.name]
            });
        }
    }
}
