import {Component, Input, ViewContainerRef} from '@angular/core';

@Component({
    selector: 'app-span-colored',
    templateUrl: './span-colored.html'
})
export class SpanColoredComponent {

    @Input() color: string;
    @Input() text: string;

    constructor(public viewContainerRef: ViewContainerRef) {

    }
}
