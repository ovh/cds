import {Injectable} from '@angular/core';
import {BehaviorSubject, Observable} from 'rxjs';
import {WorkflowNode, Workflow} from '../../model/workflow.model';

@Injectable()
export class WorkflowCoreService {

    private _linkJoinEvent: BehaviorSubject<WorkflowNode> = new BehaviorSubject(null);
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

    getLinkJoinEvent(): Observable<WorkflowNode> {
        return new Observable<WorkflowNode>(fn => this._linkJoinEvent.subscribe(fn));
    }

    linkJoinEvent(node: WorkflowNode): void {
        this._linkJoinEvent.next(node);
    }
}
