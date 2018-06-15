import {Injectable} from '@angular/core';
import {BehaviorSubject, Observable} from 'rxjs';
import {Workflow, WorkflowNode} from '../../model/workflow.model';
import {WorkflowRun} from '../../model/workflow.run.model';
import {WorkflowNodeRun} from '../../model/workflow.run.model';

@Injectable()
export class WorkflowCoreService {

    private _sideBarStatus: BehaviorSubject<boolean> = new BehaviorSubject(true);
    private _currentWorkflowRun: BehaviorSubject<WorkflowRun> = new BehaviorSubject(null);
    private _currentNodeRun: BehaviorSubject<WorkflowNodeRun> = new BehaviorSubject(null);
    private _linkJoinEvent: BehaviorSubject<WorkflowNode> = new BehaviorSubject(null);
    private _asCodeEditorEvent: BehaviorSubject<{open: boolean, save: boolean}> = new BehaviorSubject(null);
    private _previewWorkflow: BehaviorSubject<Workflow> = new BehaviorSubject(null);

    getSidebarStatus(): Observable<boolean> {
        return new Observable<boolean>(fn => this._sideBarStatus.subscribe(fn));
    }

    moveSideBar(o: boolean): void {
        this._sideBarStatus.next(o);
    }

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

    getCurrentWorkflowRun(): Observable<WorkflowRun> {
        return new Observable<WorkflowRun>(fn => this._currentWorkflowRun.subscribe(fn));
    }

    setCurrentWorkflowRun(wr: WorkflowRun): void {
        this._currentWorkflowRun.next(wr);
    }

    getCurrentNodeRun(): Observable<WorkflowNodeRun> {
        return new Observable<WorkflowNodeRun>(fn => this._currentNodeRun.subscribe(fn));
    }

    setCurrentNodeRun(wr: WorkflowNodeRun): void {
        this._currentNodeRun.next(wr);
    }
}
