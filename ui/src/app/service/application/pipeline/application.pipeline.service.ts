
import {map} from 'rxjs/operators';
import {Injectable} from '@angular/core';
import {PipelineBuild, PipelineRunRequest} from '../../../model/pipeline.model';
import {Observable} from 'rxjs';
import {Commit} from '../../../model/repositories.model';
import {HttpClient, HttpParams} from '@angular/common/http';

@Injectable()
export class ApplicationPipelineService {

    constructor(private _http: HttpClient) {
    }

    stop(key: string, appName: string, pipName: string, buildNumber: number, envName: string): Observable<boolean> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/build/' + buildNumber + '/stop';
        let params = new HttpParams();
        params = params.append('envName', envName);
        return this._http.post(url, null, {params: params}).pipe(map(res => true));
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
        let params = new HttpParams();
        params = params.append('envName', envName);
        return this._http.post<PipelineBuild>(url, null, {params: params});
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
        return this._http.post<PipelineBuild>(url, runRequest);
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
        return this._http.post<PipelineBuild>(url, runRequest);
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
                 envName: string, limit: number, status: string, branchName: string, remote: string): Observable<Array<PipelineBuild>> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/history';
        let params = new HttpParams();

        [
          {key: 'envName', value: envName},
          {key: 'limit', value: limit},
          {key: 'status', value: status},
          {key: 'branchName', value: branchName},
          {key: 'remote', value: remote}
      ].forEach((param: any) => {
          if (param.value != null) {
            params = params.append(param.key, param.value);
          }
        });

        return this._http.get<Array<PipelineBuild>>(url, {params: params});
    }

    /**
     * Get list of commits from given hash.
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param envName Environment name
     * @param hash hash
     * @returns {Observable<Array<Commit>>}
     */
    getCommits(key: string, appName: string, pipName: string, envName: string, hash: string): Observable<Array<Commit>> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/commits';
        let params = new HttpParams();
        params = params.append('envName', envName);
        params = params.append('hash', hash);
        return this._http.get<Array<Commit>>(url, {params: params});
    }

    /**
     * Get triggered pipeline from parent pipeline build info
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param buildNumber Buildnumber
     * @returns {Observable<Array<PipelineBuild>>}
     */
    getTriggeredPipeline(key: string, appName: string, pipName: string, buildNumber: number): Observable<Array<PipelineBuild>> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/build/' + buildNumber + '/triggered';
        return this._http.get<Array<PipelineBuild>>(url);
    }

    /**
     * Delete a build
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param envName Environment name
     * @param buildNumber BuildNumber
     * @returns {Observable<Boolean>}
     */
    deleteBuild(key: string, appName: string, pipName: string, envName: string, buildNumber: number): Observable<boolean> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/build/' + buildNumber;
        let params = new HttpParams();
        params = params.append('envName', envName);
        return this._http.delete(url, {params: params}).pipe(map(res => true));
    }
}
