import {Component, Input, ViewChild, ComponentFactoryResolver, ViewContainerRef} from '@angular/core';
import {SpanColoredComponent} from './span-colored/span-colored.component';
import * as JsDiff from 'diff';

@Component({
    selector: 'app-diff',
    templateUrl: './diff.html',
    styleUrls: ['./diff.scss']
})
export class DiffComponent {

    _original: string;
    @Input('original')
    set original(data: string) {
        this._original = data;
        if (this.original && this.updated) {
            this.refresh();
        }
    }
    get original() {
        return this._original;
    }

    _updated: string;
    @Input('updated')
    set updated(data: string) {
        this._updated = data;
        if (this.original && this.updated) {
            this.refresh();
        }
    }
    get updated() {
        return this._updated;
    }

    @ViewChild('diffDisplayUpdated', {read: ViewContainerRef}) diffDisplayUpdated;

    constructor(private componentFactoryResolver: ComponentFactoryResolver) {

    }

    refresh() {
      let original = this.original || '';
      if (original === 'null') {
        original = '';
      }
      let diff = JsDiff.diffWordsWithSpace(original, this.updated);

      if (!Array.isArray(diff)) {
        return;
      }
      let viewContainerRef = this.diffDisplayUpdated;
      viewContainerRef.clear();

      diff.forEach((part) => {
        let color;
        if (part.added) {
          color = '#cdffd8';
        } else if (part.removed) {
          color = '#ffeef0'
        } else {
          color = 'white';
        }
        let componentFactory = this.componentFactoryResolver.resolveComponentFactory(SpanColoredComponent)
        let componentRef = viewContainerRef.createComponent(componentFactory);
        (<SpanColoredComponent>componentRef.instance).color = color;
        (<SpanColoredComponent>componentRef.instance).text = part.value;
      });
    }
}
