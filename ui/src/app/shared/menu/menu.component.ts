import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
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

export enum Orientation {
    VERTICAL = 'VERTICAL',
    HORIZONTAL = 'HORIZONTAL'
}

@Component({
    selector: 'app-menu',
    templateUrl: './menu.html',
    styleUrls: ['./menu.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class MenuComponent implements OnInit, OnChanges, OnDestroy {
    @Input() items: Array<Item>;
    @Input() orientation: Orientation;
    @Input() withRouting: boolean;
    @Output() onSelect = new EventEmitter<Item>();

    selected: Item;
    queryParamsSub: Subscription;
    itemFromParam: string;
    Orientation = Orientation;

    constructor(
        private _route: ActivatedRoute,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.select(this.items.find(t => t.default));
        if (this.withRouting) {
            this.initQueryParamsSub();
        }
    }

    initQueryParamsSub(): void {
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

        if (this.withRouting) {
            this.initQueryParamsSub();
        }
    }

    clickSelect(item: Item) {
        if (this.withRouting) {
            this._router.navigate([], {
                relativeTo: this._route,
                queryParams: { item: item.key },
                queryParamsHandling: 'merge'
            });
        } else {
            this.select(item);
        }
    }

    select(item: Item) {
        if (item && (!this.selected || item.key !== this.selected.key)) {
            this.selected = item;
            this._cd.markForCheck();
            this.onSelect.emit(this.selected);
        }
    }
}
