import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs/Subscription';
import { Action, Usage } from '../../../../model/action.model';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { Tab } from '../../../../shared/tabs/tabs.component';

@Component({
    selector: 'app-action-show',
    templateUrl: './action.show.html',
    styleUrls: ['./action.show.scss']
})
@AutoUnsubscribe()
export class ActionShowComponent implements OnInit {
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
    ) { }

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

    }

    getUsage(): void {

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
                routerLink: ['/', 'settings', 'action', 'builtin', this.action.name]
            });
        }
    }
}
