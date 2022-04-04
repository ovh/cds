import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, TemplateRef } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs/Subscription';

export class Tab {
    title: string;
    translate_args?: {};
    icon: string;
    key: string;
    default: boolean;
    template: TemplateRef<any>;
    warningText: string;
    warningTemplate: TemplateRef<any>;
    disabled: boolean;
}

@Component({
    selector: 'app-tabs',
    templateUrl: './tabs.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class TabsComponent implements OnInit, OnChanges, OnDestroy {
    @Input() tabs: Array<Tab>;
    @Input() disableNavigation: boolean;

    @Output() onSelect = new EventEmitter<Tab>();

    selected: Tab;
    queryParamsSub: Subscription;

    constructor(private _route: ActivatedRoute, private _router: Router) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.select(this.tabs.find(t => t.default));
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['tab'] && !this.disableNavigation) {
                this.select(this.tabs.find(t => t.key === params['tab']));
            }
        });
    }

    ngOnChanges() {
        if (!this.selected) {
            let default_tab = this.tabs.find(t => t.default);
            if (default_tab) {
                this.selected = default_tab;
            } else {
                this.selected = this.tabs[0];
            }
        }
    }

    clickSelect(tab: Tab) {
        if (tab.disabled) {
            return;
        }
        if (!this.disableNavigation) {
            this._router.navigate([], {
                relativeTo: this._route,
                queryParams: { tab: tab.key },
                queryParamsHandling: 'merge'
            });
        } else {
            this.select(this.tabs.find(t => t.key === tab.key));
        }
    }

    select(tab: Tab) {
        if (tab) {
            this.selected = tab;
            this.onSelect.emit(this.selected);
        }
    }
}
