import { Directive, ElementRef, HostListener, OnInit, inject } from '@angular/core';

/**
 * Retrofits keyboard and screenreader semantics onto a non-interactive element
 * that carries a (click) handler, without changing its DOM structure or styling:
 * role="button" (unless a role is already set), tabindex="0" (unless set) and
 * Enter/Space activation. Prefer a real <button> for new code.
 */
@Directive({
    standalone: false,
    selector: '[appClickable]'
})
export class ClickableDirective implements OnInit {
    _elementRef = inject(ElementRef);

    ngOnInit(): void {
        const el = this._elementRef.nativeElement as HTMLElement;
        if (!el.hasAttribute('role')) {
            el.setAttribute('role', 'button');
        }
        if (!el.hasAttribute('tabindex')) {
            el.setAttribute('tabindex', '0');
        }
    }

    @HostListener('keydown.enter', ['$event'])
    @HostListener('keydown.space', ['$event'])
    onKeyActivate(e: Event): void {
        if (e.target !== this._elementRef.nativeElement) {
            return;
        }
        e.preventDefault();
        (this._elementRef.nativeElement as HTMLElement).click();
    }
}
