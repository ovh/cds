import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { Analysis, AnalysisRequest, AnalysisResponse } from 'app/model/analysis.model';

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

    triggerAnalysis(request: AnalysisRequest): Observable<AnalysisResponse> {
        let encodedRepo = encodeURIComponent(request.repoName);
        return this._http.post<AnalysisResponse>(`/v2/project/${request.projectKey}/vcs/${request.vcsName}/repository/${encodedRepo}/analysis`, request);
    }

    getAnalysis(projKey: string, vcsName: string, repoName: string, id: string): Observable<Analysis>  {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<Analysis>(`/v2/project/${projKey}/vcs/${vcsName}/repository/${encodedRepo}/analysis/${id}`);
    }

}
