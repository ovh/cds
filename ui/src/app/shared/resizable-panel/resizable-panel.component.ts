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
    @Input() defaultSize = null;

    @Output() onGrabbingStart = new EventEmitter<void>();
    @Output() onGrabbingEnd = new EventEmitter<void>();

    grabbing = false;

    constructor(
        private _cd: ChangeDetectorRef,
        private _renderer: Renderer2
    ) { }

    ngAfterViewInit(): void {
        if (this.direction === PanelDirection.HORIZONTAL) {
            const contentWidth = this.defaultSize ?? 600;
            this._renderer.setStyle(this.content.nativeElement, 'width', `${contentWidth - 4}px`);
            this._cd.detectChanges();
        } else {
            const contentHeight = this.defaultSize ?? 200;
            this._renderer.setStyle(this.content.nativeElement, 'height', `${contentHeight - 4}px`);
            this._cd.detectChanges();
        }
    }

    onMouseDownGrabber(): void {
        this.grabbing = true;
        this._cd.markForCheck();
        this.onGrabbingStart.emit();
    }

    @HostListener('mouseup', ['$event'])
    onMouseUpGrabber(): void {
        this.grabbing = false;
        this._cd.markForCheck();
        this.onGrabbingEnd.emit();
    }

    @HostListener('window:mousemove', ['$event'])
    onMouseMove(event: any): void {
        if (this.grabbing) {
            if (this.direction === PanelDirection.HORIZONTAL) {
                const contentWidth = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientX : window.innerWidth - event.clientX, this.defaultSize ?? 600);
                this._renderer.setStyle(this.content.nativeElement, 'width', `${contentWidth - 4}px`);
                this._cd.detectChanges();
            } else {
                const contentHeight = Math.max(this.growDirection === PanelGrowDirection.AFTER ? event.clientY : window.innerHeight - event.clientY, this.defaultSize ?? 200);
                this._renderer.setStyle(this.content.nativeElement, 'height', `${contentHeight - 4}px`);
                this._cd.detectChanges();
            }
        }
    }
}
