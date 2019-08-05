import {Injectable} from '@angular/core';
import {WNode, Workflow} from 'app/model/workflow.model';
import {BehaviorSubject, Observable} from 'rxjs';

@Injectable()
export class WorkflowCoreService {

    private _linkJoinEvent: BehaviorSubject<WNode> = new BehaviorSubject(null);
    private _asCodeEditorEvent: BehaviorSubject<{open: boolean, save: boolean}> = new BehaviorSubject(null);
    private _previewWorkflow: BehaviorSubject<Workflow> = new BehaviorSubject(null);

    getAsCodeEditor(): Observable<{open: boolean, save: boolean}> {
        return new Observable<{open: boolean, save: boolean}>(fn => this._asCodeEditorEvent.subscribe(fn));
    }

    toggleAsCodeEditor(o: {open: boolean, save: boolean}): void {
        this._asCodeEditorEvent.next(o);
    }

    getWorkflowPreview(): Observable<Workflow> {
        return new Observable<Workflow>(fn => this._previewWorkflow.subscribe(fn));
    }

    setWorkflowPreview(wf: Workflow): void {
        if (wf) {
            wf.forceRefresh = true;
            wf.previewMode = true;
        }
        this._previewWorkflow.next(wf);
    }

    getLinkJoinEvent(): Observable<WNode> {
        return new Observable<WNode>(fn => this._linkJoinEvent.subscribe(fn));
    }

    linkJoinEvent(node: WNode): void {
        this._linkJoinEvent.next(node);
    }
}
