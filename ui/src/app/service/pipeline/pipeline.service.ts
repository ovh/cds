
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Job } from 'app/model/job.model';
import { Operation } from 'app/model/operation.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Stage } from 'app/model/stage.model';
import { WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { Observable } from 'rxjs';

/**
 * Service to access Pipeline from API.
 * Only used by PipelineStore
 */
@Injectable()
export class PipelineService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the given pipeline from API
     * @param key Project unique key
     * @param pipName Pipeline Name
     */
    getPipeline(key: string, pipName: string): Observable<Pipeline> {
        let params = new HttpParams();
        params = params.append('withApplications', 'true');
        params = params.append('withWorkflows', 'true');
        params = params.append('withEnvironments', 'true');
        params = params.append('withAsCodeEvents', 'true');
        return this._http.get<Pipeline>(`/project/${key}/pipeline/${pipName}`, { params: params });
    }

    /**
     * Get the list of condition names for a given pipeline
     * @param key Project unique key
     * @param pipName Pipeline name
     * @returns {Observable<WorkflowTriggerConditionCache>}
     */
    getStageConditionsName(key: string, pipName: string): Observable<WorkflowTriggerConditionCache> {
        return this._http.get<WorkflowTriggerConditionCache>('/project/' + key + '/pipeline/' + pipName + '/stage/condition');
    }

    updateAsCode(key: string, pipeline: Pipeline, branch, message: string): Observable<Operation> {
        let params = new HttpParams();
        params = params.append('branch', branch);
        params = params.append('message', message)
        return this._http.put<Operation>(`/project/${key}/pipeline/${pipeline.name}/ascode`, pipeline, { params });
    }

    /**
     * Update the given stage
     * @param key Project unique key
     * @param pipName Pipeline Name
     * @param stage Stage to update
     * @returns {Observable<Pipeline>}
     */
    updateStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
        return this._http.put<Pipeline>('/project/' + key + '/pipeline/' + pipName + '/stage/' + stage.id, stage);
    }

    /**
     * Delete a stage
     * @param key Project unique key
     * @param pipName Pipeline Name
     * @param stage Stage to delete
     * @returns {Observable<Pipeline>}
     */
    deleteStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
        return this._http.delete<Pipeline>('/project/' + key + '/pipeline/' + pipName + '/stage/' + stage.id);
    }

    /**
     * Add a job
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param stageID Stage ID
     * @param action Job to add
     * @returns {Observable<Pipeline>}
     */
    addJob(key: string, pipName: string, stageID: number, job: Job): Observable<Pipeline> {
        return this._http.post<Pipeline>('/project/' + key + '/pipeline/' + pipName + '/stage/' + stageID + '/job', job);
    }

    /**
     * Update a job
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param stageID Stage ID
     * @param action Job to update
     * @returns {Observable<Pipeline>}
     */
    updateJob(key: string, pipName: string, stageID: number, job: Job): Observable<Pipeline> {
        let url = '/project/' + key + '/pipeline/' + pipName + '/stage/' + stageID + '/job/' + job.pipeline_action_id;
        return this._http.put<Pipeline>(url, job);
    }
}
