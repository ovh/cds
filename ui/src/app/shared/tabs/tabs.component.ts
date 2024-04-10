import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, TemplateRef } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';

export class Tab {
    title: string;
    icon: string;
    iconTheme: string;
    iconClassColor: string;
    key: string;
    default: boolean;
    template: TemplateRef<any>;
    warningText: string;
    warningTemplate: TemplateRef<any>;
    disabled: boolean;
    link: string[];
}

@Component({
    selector: 'app-tabs',
    templateUrl: './tabs.html',
    styleUrls: ['./tabs.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class TabsComponent implements OnInit, OnChanges, OnDestroy {
    @Input() tabs: Array<Tab>;
    @Input() disableNavigation: boolean;

    @Output() onSelect = new EventEmitter<Tab>();

    selected: Tab;
    queryParamsSub: Subscription;

    constructor(
        private _route: ActivatedRoute,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.select(this.tabs.find(t => t.default));
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (!this.disableNavigation) {
                if (params['tab']) {
                    this.select(this.tabs.find(t => t.key === params['tab']));
                } else {
                    this.select(this.getDefaultTab());
                }
            }
        });
    }

    ngOnChanges() {
        if (this.selected && !this.tabs.find(t => t.key === this.selected.key)) {
            delete this.selected;
        }

        if (this.selected && this._route.snapshot.queryParams['tab'] && this.selected.key !== this._route.snapshot.queryParams['tab']) {
            delete this.selected;
        }

        if (!this.selected) {
            if (!this.disableNavigation) {
                const tab = this.tabs.find(t => t.key === this._route.snapshot.queryParams['tab']);
                if (tab) {
                    this.select(tab);
                } else {
                    this.select(this.getDefaultTab());
                }
            }
        }

        this._cd.markForCheck();
    }

    getDefaultTab(): Tab {
        let defaultTab = this.tabs.find(t => t.default);
        if (defaultTab) {
            return defaultTab;
        }
        return this.tabs[0];
    }

    clickSelect(tab: Tab) {
        if (this.selected && this.selected.key === tab.key) {
            return;
        }
        if (tab.disabled) {
            return;
        }
        if (tab.link?.length > 0) {
            this._router.navigate(tab.link);
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
