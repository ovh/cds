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

    @Output() onGrabbingStart = new EventEmitter<void>();
    @Output() onGrabbingEnd = new EventEmitter<void>();

    grabbing = false;

    constructor(
        private _cd: ChangeDetectorRef,
        private _renderer: Renderer2
    ) { }

    ngAfterViewInit(): void {
        if (this.direction === PanelDirection.HORIZONTAL) {
            const contentWidth = 600;
            this._renderer.setStyle(this.content.nativeElement, 'width', `${contentWidth - 4}px`);
            this._cd.detectChanges();
        } else {
            const contentHeight = 200;
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
                const contentWidth = Math.max(window.innerWidth - event.clientX, 600);
                this._renderer.setStyle(this.content.nativeElement, 'width', `${contentWidth - 4}px`);
                this._cd.detectChanges();
            } else {
                const contentHeight = Math.max(window.innerHeight - event.clientY, 200);
                this._renderer.setStyle(this.content.nativeElement, 'height', `${contentHeight - 4}px`);
                this._cd.detectChanges();
            }
        }
    }
}
