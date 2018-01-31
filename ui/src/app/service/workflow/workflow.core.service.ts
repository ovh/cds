import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Observable} from 'rxjs/Observable';
import {WorkflowRun} from '../../model/workflow.run.model';
import {WorkflowNode, Workflow} from '../../model/workflow.model';

@Injectable()
export class WorkflowCoreService {

    private _sideBarStatus: BehaviorSubject<boolean> = new BehaviorSubject(true);
    private _currentWorkflowRun: BehaviorSubject<WorkflowRun> = new BehaviorSubject(null);
    private _linkJoinEvent: BehaviorSubject<WorkflowNode> = new BehaviorSubject(null);
    private _asCodeEditorEvent: BehaviorSubject<boolean> = new BehaviorSubject(null);
    private _previewWorkflow: BehaviorSubject<Workflow> = new BehaviorSubject(null);

    getSidebarStatus(): Observable<boolean> {
        return new Observable<boolean>(fn => this._sideBarStatus.subscribe(fn));
    }

    moveSideBar(o: boolean): void {
        this._sideBarStatus.next(o);
    }

    getAsCodeEditor(): Observable<boolean> {
        return new Observable<boolean>(fn => this._asCodeEditorEvent.subscribe(fn));
    }

    toggleAsCodeEditor(o: boolean): void {
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

    getCurrentWorkflowRun(): Observable<WorkflowRun> {
        return new Observable<WorkflowRun>(fn => this._currentWorkflowRun.subscribe(fn));
    }

    setCurrentWorkflowRun(wr: WorkflowRun): void {
        this._currentWorkflowRun.next(wr);
    }
}
