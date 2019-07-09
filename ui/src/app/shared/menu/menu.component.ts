import {
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
    styleUrls: ['./menu.scss']
})
@AutoUnsubscribe()
export class MenuComponent implements OnInit, OnChanges {
    @Input() items: Array<Item>;
    @Output() onSelect = new EventEmitter<Item>();
    selected: Item;
    queryParamsSub: Subscription;

    constructor(private _route: ActivatedRoute, private _router: Router) { }

    ngOnInit() {
        this.select(this.items.find(t => t.default));
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['item']) {
                this.select(this.items.find(t => t.key === params['item']));
            }
        });
    }

    ngOnChanges() {
        this.selected = this.items.find(t => t.default);
        if (!this.selected) {
            this.selected = this.items[0];
        }
    }

    clickSelect(item: Item) {
        this._router.navigate([], {
            relativeTo: this._route,
            queryParams: { item: item.key },
            queryParamsHandling: 'merge'
        });
    }

    select(item: Item) {
        if (item) {
            this.selected = item;
            this.onSelect.emit(this.selected);
        }
    }
}
