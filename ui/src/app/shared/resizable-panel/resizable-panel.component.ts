import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    EventEmitter,
    HostListener,
    Input,
    Output,
    Renderer2,
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
export class ResizablePanelComponent implements AfterViewInit {
    @ViewChild('grabber') grabber: ElementRef;
    @ViewChild('content') content: ElementRef;

    @Input() direction = PanelDirection.HORIZONTAL;
    @Input() growDirection = PanelGrowDirection.BEFORE;
    @Input() minSize = null;
    @Input() initialSize: number | string = null;

    @Output() onGrabbingStart = new EventEmitter<void>();
    @Output() onGrabbingEnd = new EventEmitter<number>();

    grabbing = false;
    size: number;

    constructor(
        private _cd: ChangeDetectorRef,
        private _renderer: Renderer2,
        private _elementRef: ElementRef
    ) { }

    ngAfterViewInit(): void {
        let initialSize = (this.minSize ?? (this.direction === PanelDirection.HORIZONTAL ? 600 : 200));
        if (this.initialSize) {
            if (typeof this.initialSize === 'number') {
                initialSize = this.initialSize;
            } else if ((<string>this.initialSize).endsWith('%')) {
                try {
                    const ratio = parseInt((<string>this.initialSize).replace('%', ''), 10);
                    initialSize = (ratio * (this.direction === PanelDirection.HORIZONTAL ? this._elementRef.nativeElement.parentNode.clientWidth : this._elementRef.nativeElement.parentNode.clientHeight)) / 100;
                } catch (e) { }
            }
        }
        this.size = initialSize;
        this.redraw();
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
        this.onGrabbingEnd.emit(this.size);
    }

    @HostListener('window:mousemove', ['$event'])
    onMouseMove(event: any): void {
        if (this.grabbing) {
            if (this.direction === PanelDirection.HORIZONTAL) {
                const maxSize = this._elementRef.nativeElement.parentNode.clientWidth;
                const newSize = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientX : window.innerWidth - event.clientX, this.minSize ?? 600);
                this.size = Math.min(newSize, maxSize);
            } else {
                const maxSize = this._elementRef.nativeElement.parentNode.clientHeight;
                const newSize = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientY : window.innerHeight - event.clientY, this.minSize ?? 200);
                this.size = Math.min(newSize, maxSize);
            }
            this.redraw();
        }
    }

    redraw(): void {
        if (this.direction === PanelDirection.HORIZONTAL) {
            this._renderer.setStyle(this.content.nativeElement, 'width', `${this.size - 4}px`);
            this._cd.detectChanges();
        } else {
            this._renderer.setStyle(this.content.nativeElement, 'height', `${this.size - 4}px`);
            this._cd.detectChanges();
        }
    }
}
