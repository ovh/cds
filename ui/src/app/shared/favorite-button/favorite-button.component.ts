import { EventEmitter, Component, ChangeDetectionStrategy, ChangeDetectorRef, Input, Output, OnChanges, SimpleChanges } from "@angular/core";

@Component({
    selector: 'app-favorite-button',
    templateUrl: './favorite-button.component.html',
    styleUrls: ['./favorite-button.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class FavoriteButtonComponent implements OnChanges {
    @Input() loading: boolean;
    @Input() active: boolean;
    @Output() onClick = new EventEmitter();

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngOnChanges(changes: SimpleChanges): void {
        this._cd.markForCheck();
    }

    click(): void {
        this.onClick.emit(null);
    }
}
