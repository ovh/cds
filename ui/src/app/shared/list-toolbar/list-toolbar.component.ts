import { AfterViewInit, ChangeDetectionStrategy, Component, ElementRef, EventEmitter, Input, Output, ViewChild } from '@angular/core';

@Component({
    standalone: false,
    selector: 'app-list-toolbar',
    templateUrl: './list-toolbar.html',
    styleUrls: ['./list-toolbar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ListToolbarComponent implements AfterViewInit {
    @ViewChild('searchInput') searchInput: ElementRef<HTMLInputElement>;

    @Input() searchPlaceholder: string = 'Search';
    @Input() searchValue: string = '';
    @Input() itemLabel: string = '';
    @Input() count: number;
    @Input() total: number;
    @Input() addLabel: string;
    @Input() autofocus: boolean = true;

    @Output() searchChange: EventEmitter<string> = new EventEmitter();
    @Output() add: EventEmitter<void> = new EventEmitter();

    ngAfterViewInit(): void {
        if (this.autofocus) {
            setTimeout(() => this.searchInput?.nativeElement.focus({ preventScroll: true }));
        }
    }
}
