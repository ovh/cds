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
    Renderer2,
    SimpleChanges,
    ViewChild
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

export enum PanelDirection {
    HORIZONTAL = 'horizontal',
    VERTICAL = 'vertical'
}

export enum PanelGrowDirection {
    BEFORE = 'before',
    AFTER = 'after'
}

@Component({
    selector: 'app-resizable-panel',
    templateUrl: './resizable-panel.html',
    styleUrls: ['./resizable-panel.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ResizablePanelComponent implements AfterViewInit, OnChanges {
    @ViewChild('grabber') grabber: ElementRef;
    @ViewChild('content') content: ElementRef;

    @Input() direction = PanelDirection.HORIZONTAL;
    @Input() growDirection = PanelGrowDirection.BEFORE;
    @Input() minSize: number = null;
    @Input() initialSize: number | string = null;

    @Output() onGrabbingStart = new EventEmitter<void>();
    @Output() onGrabbingEnd = new EventEmitter<string>();

    grabbing = false;
    sizePixels: number;
    sizePercents: number;

    constructor(
        private _cd: ChangeDetectorRef,
        private _renderer: Renderer2,
        private _elementRef: ElementRef
    ) { }

    ngOnChanges(changes: SimpleChanges): void {
        this.init();
    }

    ngAfterViewInit(): void {
        setTimeout(() => this.init(), 1);
    }

    init() {
        const rect = this._elementRef.nativeElement.parentNode.getBoundingClientRect();
        let initialSize: number = 0;
        if (this.initialSize) {
            if (typeof this.initialSize === 'number') {
                initialSize = this.initialSize;
            } else if ((<string>this.initialSize).endsWith('%')) {
                try {
                    const ratio = parseFloat((<string>this.initialSize).replace('%', ''));
                    initialSize = (ratio * (this.direction === PanelDirection.HORIZONTAL ? rect.width : rect.height)) / 100;
                } catch (e) { }
            }
        }
        const minSize = (this.minSize ?? (this.direction === PanelDirection.HORIZONTAL ? 600 : 200));
        const maxSize = (this.direction === PanelDirection.HORIZONTAL ? rect.width : rect.height) - 10;
        if (initialSize < minSize) { initialSize = minSize; }
        if (initialSize > maxSize) { initialSize = maxSize; }
        this.sizePixels = initialSize;
        this.computeSizePercents();
        this.redraw();
    }

    computeSizePercents(): void {
        const rect = this._elementRef.nativeElement.parentNode.getBoundingClientRect();
        this.sizePercents = (this.sizePixels / (this.direction === PanelDirection.HORIZONTAL ? rect.width : rect.height)) * 100;
    }

    onMouseDownGrabber(): void {
        this.grabbing = true;
        this._cd.detectChanges();
        this.onGrabbingStart.emit();
    }

    @HostListener('window:mouseup', ['$event'])
    onMouseUpGrabber(): void {
        if (!this.grabbing) {
            return;
        }
        this.grabbing = false;
        this._cd.detectChanges();
        this.onGrabbingEnd.emit(this.sizePercents + '%');
    }

    @HostListener('window:mousemove', ['$event'])
    onMouseMove(event: any): void {
        if (this.grabbing) {
            const rect = this._elementRef.nativeElement.parentNode.getBoundingClientRect();
            if (this.direction === PanelDirection.HORIZONTAL) {
                const maxSize = rect.width - 10;
                const newSize = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientX - rect.left : rect.right - event.clientX, this.minSize ?? 600);
                this.sizePixels = Math.min(newSize, maxSize);
                this.computeSizePercents();
            } else {
                const maxSize = rect.height - 10;
                const newSize = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientY - rect.top : rect.bottom - event.clientY, this.minSize ?? 200);
                this.sizePixels = Math.min(newSize, maxSize);
                this.computeSizePercents();
            }
            this.redraw();
        }
    }

    @HostListener('window:resize', ['$event'])
    onResize(event: any) {
        const rect = this._elementRef.nativeElement.parentNode.getBoundingClientRect();
        let size = (this.sizePercents * (this.direction === PanelDirection.HORIZONTAL ? rect.width : rect.height)) / 100;
        const minSize = (this.minSize ?? (this.direction === PanelDirection.HORIZONTAL ? 600 : 200));
        const maxSize = this.direction === PanelDirection.HORIZONTAL ? rect.width : rect.height;
        if (size < minSize) { size = minSize; }
        if (size > maxSize) { size = maxSize; }
        this.sizePixels = size;
        this.redraw();
    }

    redraw(): void {
        if (!this.content) { return; }
        if (this.direction === PanelDirection.HORIZONTAL) {
            this._renderer.setStyle(this.content.nativeElement, 'width', `${this.sizePixels}px`);
            this._cd.detectChanges();
        } else {
            this._renderer.setStyle(this.content.nativeElement, 'height', `${this.sizePixels}px`);
            this._cd.detectChanges();
        }
    }
}
