import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnInit,
    Output
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs/Subscription';

export class Item {
    translate: string;
    key: string;
    default: boolean;
}

@Component({
    selector: 'app-menu',
    templateUrl: './menu.html',
    styleUrls: ['./menu.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class MenuComponent implements OnInit, OnChanges {
    @Input() items: Array<Item>;
    @Output() onSelect = new EventEmitter<Item>();
    selected: Item;
    queryParamsSub: Subscription;
    itemFromParam: string;

    constructor(
        private _route: ActivatedRoute,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        this.select(this.items.find(t => t.default));
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['item']) {
                this.itemFromParam = params['item'];
                this.select(this.items.find(t => t.key === this.itemFromParam));
            }
        });
    }

    ngOnChanges() {
        let newSelected = this.items.find(t => t.key === this.itemFromParam);
        if (!newSelected) {
            newSelected = this.items.find(t => t.default);
        }
        if (!newSelected && this.items.length > 0) {
            newSelected = this.items[0];
        }
        this.select(newSelected);
    }

    clickSelect(item: Item) {
        this._router.navigate([], {
            relativeTo: this._route,
            queryParams: { item: item.key },
            queryParamsHandling: 'merge'
        });
    }

    select(item: Item) {
        if (item && (!this.selected || item.key !== this.selected.key)) {
            this.selected = item;
            this._cd.markForCheck();
            this.onSelect.emit(this.selected);
        }
    }
}
