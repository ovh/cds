import { Injectable } from '@angular/core';

@Injectable()
export class WorkflowStore {

    WORKFLOW_ORIENTATION_KEY = 'CDS-WORKFLOW-ORIENTATION';

    constructor() { }

    getDirection(key: string, name: string) {
        let o = localStorage.getItem(this.WORKFLOW_ORIENTATION_KEY);
        if (o) {
            let j = JSON.parse(o);
            if (j[key + '-' + name]) {
                return j[key + '-' + name];
            }
        }
        return 'LR';
    }

    setDirection(key: string, name: string, o: string) {
        if (!key || !name) {
            return;
        }
        let ls = localStorage.getItem(this.WORKFLOW_ORIENTATION_KEY);
        let j = {};
        if (ls) {
            j = JSON.parse(ls);
        }
        j[key + '-' + name] = o;
        localStorage.setItem(this.WORKFLOW_ORIENTATION_KEY, JSON.stringify(j));
    }
}
