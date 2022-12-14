import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { FlatNodeItem } from 'app/shared/tree/tree.component';

export class AnalysisEvent {
    vcsID: string
    repoID: string
    analysisID: string
    status: string

    constructor(vcsID: string, repoID: string, analysisID: string, status: string) {
        this.vcsID = vcsID;
        this.repoID = repoID;
        this.analysisID = analysisID;
        this.status = status;
    }
}

@Injectable()
export class AnalysisService {

    private _analysis: BehaviorSubject<AnalysisEvent> = new BehaviorSubject(null);

    constructor(private _http: HttpClient) { }

    getObservable(): Observable<AnalysisEvent> {
        return new Observable<AnalysisEvent>(fn => this._analysis.subscribe(fn));
    }

    sendEvent(event: AnalysisEvent): void {
        if (!event.analysisID || !event.repoID || !event.vcsID || !event.status) {
            return;
        }
        this._analysis.next(event);
    }

}
