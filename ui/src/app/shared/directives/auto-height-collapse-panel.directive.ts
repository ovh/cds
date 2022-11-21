import { ChangeDetectorRef, Directive, ElementRef, Input, OnChanges } from '@angular/core';

const panelheaderSize: number = 46;

@Directive({
    selector: '[appAutoHeightCollapsePanel]'
})
export class AutoHeightCollapsePanelDirective implements OnChanges {
    @Input('appAutoHeightCollapsePanel') panels: boolean[];
    @Input() panelIndex: number;

    constructor(private element: ElementRef, private cd: ChangeDetectorRef) {
    }

    ngOnChanges () {
        this.doAutoSize();
    }

    private doAutoSize() {
        if (this.panels?.length == 0) {
            return;
        }
        if (this.panels.length -1 < this.panelIndex) {
            return;
        }
        // Panel close
        if (!this.panels[this.panelIndex]) {
            this.element.nativeElement.style.height = 'auto';
            return;
        }

        // Get parent size
        if (!this.element.nativeElement.parentElement) {
            return;
        }
        let parentHeight = this.element.nativeElement.parentElement.offsetHeight;

        let nbOpenedPanel = this.panels.filter( opened => opened).length
        let percentHeight = 100 / nbOpenedPanel;
        percentHeight = percentHeight - 5*(this.panels.length - nbOpenedPanel);

        let height = parentHeight / 100 * percentHeight;
        height -= 46; // remove button height


        this.element.nativeElement.style.height = height + 'px';

        // Update content height
        if (this.element.nativeElement.children.length !== 2) {
            return;
        }
        this.element.nativeElement.children[1].style.height = height + 'px';
    }
}
