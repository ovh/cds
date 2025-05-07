
import { Injectable } from '@angular/core';

@Injectable()
export class ProjectStore {

    private WORKFLOW_VIEW_MODE = 'CDS-WORKFLOW-VIEW-MODE';

    constructor() { }

    getWorkflowViewMode(key: string): 'blocs' | 'labels' | 'lines' {
        let o = localStorage.getItem(this.WORKFLOW_VIEW_MODE);
        if (o) {
            let j = JSON.parse(o);
            if (j[key]) {
                return j[key];
            }
        }
        return 'blocs';
    }

    setWorkflowViewMode(key: string, viewMode: 'blocs' | 'labels' | 'lines') {
        let ls = localStorage.getItem(this.WORKFLOW_VIEW_MODE);
        let j = {};
        if (ls) {
            j = JSON.parse(ls);
        }
        j[key] = viewMode;
        localStorage.setItem(this.WORKFLOW_VIEW_MODE, JSON.stringify(j));
    }
}
