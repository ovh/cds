/**
 * FROM https://github.com/NagRock/ngx-auto-scroll/blob/master/src/ngx-auto-scroll.directive.ts
 */
import {AfterContentInit, Directive, ElementRef, HostListener, Input, OnDestroy} from "@angular/core";

@Directive({
    // eslint-disable-next-line @angular-eslint/directive-selector
    selector: "[ngx-auto-scroll]",
})
export class NgxAutoScrollDirective implements AfterContentInit, OnDestroy {

    @Input() public lockYOffset: number = 10;
    @Input() public observeAttributes: string = "false";

    private nativeElement: HTMLElement;
    private _isLocked: boolean = false;
    private mutationObserver: MutationObserver;

    constructor(element: ElementRef) {
        this.nativeElement = element.nativeElement;
    }

    public getObserveAttributes(): boolean {
        return this.observeAttributes !== "" && this.observeAttributes.toLowerCase() !== "false";
    }

    public ngAfterContentInit(): void {
        this.mutationObserver = new MutationObserver(() => {
            if (!this._isLocked) {
                this.scrollDown();
            }
        });
        this.mutationObserver.observe(this.nativeElement, {
            childList: true,
            subtree: true,
            attributes: this.getObserveAttributes(),
        });
    }

    public ngOnDestroy(): void {
        this.mutationObserver.disconnect();
    }

    public forceScrollDown(): void {
        this.scrollDown();
    }

    public isLocked(): boolean {
        return this._isLocked;
    }

    private scrollDown(): void {
        this.nativeElement.scrollTop = this.nativeElement.scrollHeight;
    }

    @HostListener("scroll")
    private scrollHandler(): void {
        const scrollFromBottom = this.nativeElement.scrollHeight - this.nativeElement.scrollTop - this.nativeElement.clientHeight;
        this._isLocked = scrollFromBottom > this.lockYOffset;
    }
}
