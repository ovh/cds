import {Injectable} from '@angular/core';
import {Http, RequestOptions, URLSearchParams} from '@angular/http';
import {PipelineBuild, PipelineRunRequest, BuildResult} from '../../../model/pipeline.model';
import {Observable} from 'rxjs/Rx';

@Injectable()
export class ApplicationPipelineService {

    constructor(private _http: Http) {
    }

    stop(key: string, appName: string, pipName: string, buildNumber: number, envName: string): Observable<boolean> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/build/' + buildNumber + '/stop';
        let options = new RequestOptions();
        options.params = new URLSearchParams();
        options.params.set('envName', envName);
        return this._http.post(url, null, options).map(res => true);
    }

    /**
     * Restart a build
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param buildNumber BuildNumber to restart
     * @param envName Environment name
     * @returns {Observable<PipelineBuild>}
     */
    runAgain(key: string, appName: string, pipName: string, buildNumber: number, envName: string): Observable<PipelineBuild> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/build/' + buildNumber + '/restart';
        let options = new RequestOptions();
        options.params = new URLSearchParams();
        options.params.set('envName', envName);
        return this._http.post(url, null, options).map(res => res.json());
    }

    /**
     * Run a pipeline
     * @param key Project Unique key
     * @param appName Application name
     * @param pipName Pipeline Name
     * @param runRequest Request to API
     * @returns {Observable<PipelineBuild>}
     */
    run(key: string, appName: string, pipName: string, runRequest: PipelineRunRequest): Observable<PipelineBuild> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/run';
        return this._http.post(url, runRequest).map(res => res.json());
    }

    /**
     * Rollback application pipeline to previous version
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param runRequest Request to API
     * @returns {Observable<PipelineBuild>}
     */
    rollback(key: string, appName: string, pipName: string, runRequest: PipelineRunRequest): Observable<PipelineBuild> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/rollback';
        return this._http.post(url, runRequest).map(res => res.json());
    }

    /**
     * Get application pipeline history
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param envName Environment filter
     * @param limit Number of result
     * @param status Status filter
     * @param branchName Branch filter
     * @param stage Give result with stage or not
     * @returns {Observable<Array<PipelineBuild>>}
     */
    buildHistory(key: string, appName: string, pipName: string,
                 envName: string, limit: number, status: string, branchName: string): Observable<Array<PipelineBuild>> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/history';
        let options = new RequestOptions();
        options.params = new URLSearchParams();
        options.params.set('envName', envName);
        options.params.set('limit', String(limit));
        options.params.set('status', status);
        options.params.set('branchName', branchName);
        return this._http.get(url, options).map(res => res.json());
    }
}
