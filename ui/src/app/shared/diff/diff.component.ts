import {Component, Input, ViewChild, OnInit, ComponentFactoryResolver, ViewContainerRef} from '@angular/core';
import {SpanColoredComponent} from './span-colored/span-colored.component';
import * as JsDiff from 'diff';

@Component({
    selector: 'app-diff',
    templateUrl: './diff.html',
    styleUrls: ['./diff.scss']
})
export class DiffComponent implements OnInit {

    @Input() original: string;
    @Input() updated: string;
    @ViewChild('diffDisplayUpdated', {read: ViewContainerRef}) diffDisplayUpdated;

    constructor(private componentFactoryResolver: ComponentFactoryResolver) {

    }

    ngOnInit() {
      let original = this.original || '';
      if (original === 'null') {
        original = '';
      }
      let diff = JsDiff.diffWordsWithSpace(original, this.updated)
      let fragment = document.createDocumentFragment();

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
