import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, OnInit, Output } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs/Subscription';

export class Tab {
    translate: string;
    icon: string;
    key: string;
    default: boolean;
}

@Component({
    selector: 'app-tabs',
    templateUrl: './tabs.html',
    styleUrls: ['./tabs.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class TabsComponent implements OnInit, OnChanges {
    @Input() tabs: Array<Tab>;
    @Input() warnings: { [key: string]: Array<Tab> };
    @Output() onSelect = new EventEmitter<Tab>();
    selected: Tab;
    queryParamsSub: Subscription;

    constructor(private _route: ActivatedRoute, private _router: Router) { }

    ngOnInit() {
        this.select(this.tabs.find(t => t.default));
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['tab']) {
                this.select(this.tabs.find(t => t.key === params['tab']));
            }
        });
    }

    ngOnChanges() {
        this.selected = this.tabs.find(t => t.default);
        if (!this.selected) {
            this.selected = this.tabs[0];
        }
    }

    clickSelect(tab: Tab) {
        this._router.navigate([], {
            relativeTo: this._route,
            queryParams: { tab: tab.key },
            queryParamsHandling: 'merge'
        });
    }

    select(tab: Tab) {
        if (tab) {
            this.selected = tab;
            this.onSelect.emit(this.selected);
        }
    }
}
