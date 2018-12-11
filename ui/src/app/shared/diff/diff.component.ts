import { Component, ComponentFactoryResolver, Input, OnChanges, ViewChild, ViewContainerRef } from '@angular/core';
import * as JsDiff from 'diff';

export class Part {
    color: string;
    text: string;
}

@Component({
    selector: 'app-diff',
    templateUrl: './diff.html',
    styleUrls: ['./diff.scss']
})
export class DiffComponent implements OnChanges {
    @ViewChild('diffDisplayUpdated', { read: ViewContainerRef }) diffDisplayUpdated;
    @Input() original: string;
    @Input() updated: string;
    diff: Array<Part>;

    constructor(private componentFactoryResolver: ComponentFactoryResolver) { }

    ngOnChanges() {
        if (this.original && this.updated) {
            this.refresh();
        }
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

        this.diff = diff.map(part => {
            let color;
            if (part.added) {
                color = '#cdffd8';
            } else if (part.removed) {
                color = '#ffeef0'
            } else {
                color = 'white';
            }
            return <Part>{
                text: part.value,
                color
            }
        });
    }
}
