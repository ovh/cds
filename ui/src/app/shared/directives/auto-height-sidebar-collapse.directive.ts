import { ChangeDetectorRef, Directive, ElementRef, Input, OnChanges } from '@angular/core';

@Directive({
    selector: '[appAutoHeightCollapsePanel]'
})
export class AutoHeightSidebarCollapseDirective implements OnChanges {

    @Input('appAutoHeightCollapsePanel') panels: boolean[];
    @Input() collapsed: boolean

    constructor(private element: ElementRef, private cd: ChangeDetectorRef) {
    }

    ngOnChanges () {
        this.doAutoSize();
    }

    private doAutoSize() {
        if (this.panels?.length == 0) {
            return;
        }

        // Get parent size
        if (!this.element.nativeElement.parentElement) {
            return;
        }
        let parentHeight = this.element.nativeElement.parentElement.offsetHeight;
        let collapseFullHeight = parentHeight;
        this.element.nativeElement.style.height = collapseFullHeight + 'px';

        // Remove height from panel headers
        let panelContentHeight = collapseFullHeight - 46 * this.panels.length;

        // Search how many opened panels
        let openedPanels = this.panels.filter(p => p).length;
        panelContentHeight = panelContentHeight / openedPanels;

        for (let i=0; i<this.element.nativeElement.children.length; i++) {
            if (this.panels[i]) {
                let elt = this.element.nativeElement.children[i];
                if (elt.children.length == 2) {
                    // set height on panel content
                    elt.children[1].style.height = panelContentHeight + 'px';
                }
            }
        }
    }
}
