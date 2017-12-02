import {Component, Input, ViewChild, OnInit} from '@angular/core';
import * as JsDiff from 'diff';

@Component({
    selector: 'app-diff',
    templateUrl: './diff.html',
    styleUrls: ['./diff.scss']
})
export class DiffComponent implements OnInit {

    @Input() original: string;
    @Input() updated: string;
    @ViewChild('diffDisplayUpdated') diffDisplayUpdated;

    constructor() {

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

      diff.forEach((part) => {
        let color;
        if (part.added) {
          color = '#cdffd8';
        } else if (part.removed) {
          color = '#ffeef0'
        } else {
          color = 'white';
        }
        let span = document.createElement('span');
        span.style['background-color'] = color;
        span.appendChild(document.createTextNode(part.value));
        fragment.appendChild(span);
      });

      this.diffDisplayUpdated.nativeElement.appendChild(fragment);
    }
}
