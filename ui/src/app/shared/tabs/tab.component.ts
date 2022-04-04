import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    Output,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import { AutoUnsubscribe } from '../decorator/autoUnsubscribe';
import { Tab } from './tabs.component';

@Component({
    selector: 'app-tab',
    templateUrl: './tab.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class TabComponent implements AfterViewInit, OnChanges {
    @ViewChild('templateSibling', { read: ViewContainerRef }) templateSibling: ViewContainerRef;
    @Input() tab: Tab;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngAfterViewInit(): void {
        this.drawTemplate();
    }

    ngOnChanges(): void {
        this.drawTemplate();
    }

    drawTemplate(): void {
        if (this.tab.template && this.templateSibling) {
            this.templateSibling.clear();
            this.templateSibling.createEmbeddedView(this.tab.template);
            this._cd.markForCheck();
        }
    }
}
