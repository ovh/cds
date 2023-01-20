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
    @Input() initialSize = null;

    @Output() onGrabbingStart = new EventEmitter<void>();
    @Output() onGrabbingEnd = new EventEmitter<number>();

    grabbing = false;
    size: number;

    constructor(
        private _cd: ChangeDetectorRef,
        private _renderer: Renderer2
    ) { }

    ngAfterViewInit(): void {
        if (this.direction === PanelDirection.HORIZONTAL) {
            this.size = this.initialSize ?? (this.minSize ?? 600);
        } else {
            this.size = this.initialSize ?? (this.minSize ?? 200);
        }
        this.redraw();
    }

    onMouseDownGrabber(): void {
        this.grabbing = true;
        this._cd.detectChanges();
        this.onGrabbingStart.emit();
    }

    @HostListener('mouseup', ['$event'])
    onMouseUpGrabber(): void {
        this.grabbing = false;
        this._cd.detectChanges();
        this.onGrabbingEnd.emit(this.size);
    }

    @HostListener('window:mousemove', ['$event'])
    onMouseMove(event: any): void {
        if (this.grabbing) {
            if (this.direction === PanelDirection.HORIZONTAL) {
                this.size = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientX : window.innerWidth - event.clientX, this.minSize ?? 600);
            } else {
                this.size = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientY : window.innerHeight - event.clientY, this.minSize ?? 200);
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
