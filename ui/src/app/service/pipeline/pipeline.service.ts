import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {Pipeline} from '../../model/pipeline.model';
import {Application} from '../../model/application.model';
import {GroupPermission} from '../../model/group.model';
import {Stage} from '../../model/stage.model';
import {Job} from '../../model/job.model';
import {Parameter} from '../../model/parameter.model';

/**
 * Service to access Pipeline from API.
 * Only used by PipelineStore
 */
@Injectable()
export class PipelineService {

    constructor(private _http: Http) {
    }

    /**
     * Get the given pipeline from API
     * @param key Project unique key
     * @param pipName Pipeline Name
     */
    getPipeline(key: string, pipName: string): Observable<Pipeline> {
        return this._http.get('/project/' + key + '/pipeline/' + pipName).map(res => res.json());
    }

    /**
     * Update the given pipeline
     * @param key Project unique key
     * @param pipeline Pipeline to update
     * @returns {Observable<Pipeline>}
     */
    updatePipeline(key: string, oldName: string, pipeline: Pipeline): Observable<Pipeline> {
        return this._http.put('/project/' + key + '/pipeline/' + oldName, pipeline).map(res => res.json());
    }

    /**
     * Delete a pipeline
     * @param key Project unique key
     * @param pipName Pipeline name to delete
     * @returns {Observable<boolean>}
     */
    deletePipeline(key: string, pipName: string): Observable<boolean> {
        return this._http.delete('/project/' + key + '/pipeline/' + pipName).map(() => {
            return true;
        });
    }

    /**
     * Create a new pipeline in the given project
     * @param key Project Unique Key
     * @param pipeline Pipeline to create
     * @returns {Observable<Pipeline>}
     */
    createPipeline(key: string, pipeline: Pipeline): Observable<Pipeline> {
        return this._http.post('/project/' + key + '/pipeline', pipeline).map(res => res.json());
    }

    /**
     * Get the list of applications that use the given pipeline
     * @param key Project unique key
     * @param pipName Pipeline name
     * @returns {Observable<Application[]>}
     */
    getApplications(key: string,  pipName: string): Observable<Application[]> {
        return this._http.get('/project/' + key + '/pipeline/' + pipName + '/application').map(res => res.json());
    }

    /**
     * Get the list of pipeline type
     * @returns {Observable<Array<string>>}
     */
    getPipelineTypes(): Observable<Array<string>> {
        return this._http.get('/pipeline/type').map(res => res.json());
    }

    /**
     * Insert a new Stage
     * @param key Project unique key
     * @param pipName Pipeline Name
     * @param stage Stage to add
     * @returns {Observable<Pipeline>}
     */
    insertStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
        return this._http.post('/project/' + key + '/pipeline/' + pipName + '/stage', stage).map(res => res.json());
    }

    /**
     * Update the given stage
     * @param key Project unique key
     * @param pipName Pipeline Name
     * @param stage Stage to update
     * @returns {Observable<Pipeline>}
     */
    updateStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
        return this._http.put('/project/' + key + '/pipeline/' + pipName + '/stage/' + stage.id, stage).map(res => res.json());
    }

    /**
     * Delete a stage
     * @param key Project unique key
     * @param pipName Pipeline Name
     * @param stage Stage to delete
     * @returns {Observable<Pipeline>}
     */
    deleteStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
        return this._http.delete('/project/' + key + '/pipeline/' + pipName + '/stage/' + stage.id).map(res => res.json());
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
        return this._http.post('/project/' + key + '/pipeline/' + pipName + '/stage/' + stageID + '/job', job).map(res => res.json());
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
        return this._http.put(url, job).map(res => res.json());
    }

    /**
     * Delete a job
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param stageID Stage ID
     * @param action Job to delete
     * @returns {Observable<Pipeline>}
     */
    removeJob(key: string, pipName: string, stageID: number, job: Job): Observable<Pipeline> {
        let url = '/project/' + key + '/pipeline/' + pipName + '/stage/' + stageID + '/job/' + job.pipeline_action_id;
        return this._http.delete(url).map(res => res.json());
    }

    /**
     * Add a permission on the pipeline.
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param gp Permission to add
     * @returns {Observable<Pipeline>}
     */
    addPermission(key: string, pipName: string, gp: GroupPermission): Observable<Pipeline> {
        return this._http.post('/project/' + key + '/pipeline/' + pipName + '/group', gp).map(res => res.json());
    }

    /**
     * Update a permission.
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param gp Permission to update
     * @returns {Observable<Pipeline>}
     */
    updatePermission(key: string, pipName: string, gp: GroupPermission): Observable<Pipeline> {
        return this._http.put('/project/' + key + '/pipeline/' + pipName + '/group/' + gp.group.name, gp).map(res => res.json());
    }

    /**
     * Delete a permission.
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param gp Permission to delete
     * @returns {Observable<Pipeline>}
     */
    removePermission(key: string, pipName: string, gp: GroupPermission): Observable<Pipeline> {
        return this._http.delete('/project/' + key + '/pipeline/' + pipName + '/group/' + gp.group.name).map(res => res.json());
    }

    /**
     * Add a parameter on the pipeline.
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param param Parameter to add
     * @returns {Observable<Pipeline>}
     */
    addParameter(key: string, pipName: string, param: Parameter): Observable<Pipeline> {
        return this._http.post('/project/' + key + '/pipeline/' + pipName + '/parameter/' + param.name, param).map(res => res.json());
    }

    /**
     * Update a parameter on the pipeline.
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param param Parameter to update
     * @returns {Observable<Pipeline>}
     */
    updateParameter(key: string, pipName: string, param: Parameter): Observable<Pipeline> {
        return this._http.put('/project/' + key + '/pipeline/' + pipName + '/parameter/' + param.name, param).map(res => res.json());
    }

    /**
     * Remove a parameter from the pipeline.
     * @param key Project unique key
     * @param pipName Pipeline name
     * @param param Parameter to remove
     * @returns {Observable<Pipeline>}
     */
    removeParameter(key: string, pipName: string, param: Parameter): Observable<Pipeline> {
        return this._http.delete('/project/' + key + '/pipeline/' + pipName + '/parameter/' + param.name).map(res => res.json());
    }

}
