import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    EventEmitter,
    HostListener,
    Input,
    OnChanges,
    Output,
    ViewChild
} from '@angular/core';

@Component({
    selector: 'app-pagination',
    templateUrl: './pagination.html',
    styleUrls: ['./pagination.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class PaginationComponent implements OnChanges, AfterViewInit {
    @ViewChild('paginationWrapper') paginationWrapper: ElementRef;
    @Input() collectionSize: number;
    @Input() pageSize: number;
    @Input() page: number;
    @Output() pageChange = new EventEmitter<number>();

    maxSize = 10;

    constructor(public _cd: ChangeDetectorRef) { }

    ngAfterViewInit() {
        this.resize();
    }

    ngOnChanges() {
        this.resize();
    }

    onPageChange(n: number) {
        this.pageChange.emit(n);
    }

    @HostListener('window:resize', ['$event'])
    onResize(event) {
        this.resize();
    }

    resize() {
        if (this.paginationWrapper) {
            let wrapperWidth = this.paginationWrapper.nativeElement.clientWidth;
            // 75px is approximately the size of a button, 4 buttons are removed (left, right and ellipsis)
            let maxPage = Math.trunc(wrapperWidth / 65) - 4;
            this.maxSize = maxPage < 10 ? maxPage : 10;
            this.maxSize = this.maxSize < 1 ? 1 : this.maxSize;
            this._cd.detectChanges();
        }
    }
}
