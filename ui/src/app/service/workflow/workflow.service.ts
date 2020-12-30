import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Operation } from 'app/model/operation.model';
import {
    BuildResult,
    CDNLine,
    CDNLinesResponse,
    CDNLogLink,
    CDNLogLinks,
    CDNLogsLines,
    ServiceLog,
    SpawnInfo
} from 'app/model/pipeline.model';
import { WorkflowRetentoinDryRunResponse } from 'app/model/purge.model';
import { Workflow, WorkflowPull, WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { Observable, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';

@Injectable()
export class WorkflowService {
    constructor(private _http: HttpClient) { }

    getWorkflow(projectKey: string, workflowName: string): Observable<Workflow> {
        let params = new HttpParams();
        params = params.append('withUsage', 'true');
        params = params.append('withAudits', 'true');
        params = params.append('withTemplate', 'true');
        params = params.append('withAsCodeEvents', 'true');
        return this._http.get<Workflow>(`/project/${projectKey}/workflows/${workflowName}`, { params });
    }

    pullWorkflow(projectKey: string, workflowName: string): Observable<WorkflowPull> {
        let params = new HttpParams();
        params = params.append('json', 'true');
        return this._http.get<WorkflowPull>(`/project/${projectKey}/pull/workflows/${workflowName}`, { params });
    }

    getTriggerCondition(projectKey: string, workflowName: string, nodeID: number): Observable<WorkflowTriggerConditionCache> {
        let params = new HttpParams();
        if (nodeID) {
            params = params.append('nodeID', nodeID.toString());
        }
        return this._http.get<WorkflowTriggerConditionCache>(
            `/project/${projectKey}/workflows/${workflowName}/triggers/condition`, { params });
    }

    getTriggerHookCondition(projectKey: string, workflowName: string): Observable<WorkflowTriggerConditionCache> {
        return this._http.get<WorkflowTriggerConditionCache>(
            `/project/${projectKey}/workflows/${workflowName}/hook/triggers/condition`);
    }

    updateAsCode(projectKey: string, workflowName: string, branch: string, message: string, wf: Workflow): Observable<Operation> {
        let params = new HttpParams();
        params = params.append('branch', branch);
        params = params.append('message', message);
        if (!wf) {
            params = params.append('migrate', 'true');
        }
        return this._http.post<Operation>(`/project/${projectKey}/workflows/${workflowName}/ascode`, wf, { params });
    }

    updateRunNumber(projectKey: string, workflowName: string, runNumber: number): Observable<null> {
        return this._http.post<null>(
            `/project/${projectKey}/workflows/${workflowName}/runs/num`,
            { num: runNumber }
        );
    }

    getStepLog(projectKey: string, workflowName: string, nodeRunID: number, jobRunID: number, stepOrder: number): Observable<BuildResult> {
        return this._http.get<BuildResult>(`/project/${projectKey}/workflows/${workflowName}/nodes/${nodeRunID}/job/${jobRunID}/step/${stepOrder}/log`);
    }

    getStepLink(projectKey: string, workflowName: string, nodeRunID: number,
        jobRunID: number, stepOrder: number): Observable<CDNLogLink> {
        return this._http.get<CDNLogLink>(`/project/${projectKey}/workflows/${workflowName}/nodes/${nodeRunID}/job/${jobRunID}/step/${stepOrder}/link`);
    }

    getAllStepsLinks(projectKey: string, workflowName: string, nodeRunID: number, jobRunID: number): Observable<CDNLogLinks> {
        return this._http.get<CDNLogLinks>(`/project/${projectKey}/workflows/${workflowName}/nodes/${nodeRunID}/job/${jobRunID}/links`);
    }

    getLogsLinesCount(links: CDNLogLinks, limit: number): Observable<Array<CDNLogsLines>> {
        let params = new HttpParams();
        links.datas.map(l => l.api_ref).forEach(ref => {
            params = params.append('apiRefHash', ref)
        })
        params.append('limit', limit.toString());
        return this._http.get<Array<CDNLogsLines>>(`./cdscdn/item/${links.item_type}/lines`, { params })
    }

    getLogLines(link: CDNLogLink, params?: { [key: string]: string }): Observable<CDNLinesResponse> {
        return this._http.get(`./cdscdn/item/${link.item_type}/${link.api_ref}/lines`, { params, observe: 'response' })
            .pipe(map(res => {
                let headers: HttpHeaders = res.headers;
                return <CDNLinesResponse>{
                    totalCount: parseInt(headers.get('X-Total-Count'), 10),
                    lines: res.body as Array<CDNLine>,
                    found: true
                };
            }),
            catchError(err => {
                if (err.status === 404) {
                    return of(<CDNLinesResponse>{ lines: [], totalCount: 0, found: false });
                }
            }));
    }

    getLogDownload(link: CDNLogLink): Observable<string> {
        return this._http.get(`./cdscdn/item/${link.item_type}/${link.api_ref}/download`, { responseType: 'text' });
    }

    getServiceLog(projectKey: string, workflowName: string, nodeRunID: number,
        jobRunID: number, serviceName: string): Observable<ServiceLog> {
        return this._http.get<ServiceLog>(`/project/${projectKey}/workflows/${workflowName}/nodes/${nodeRunID}/job/${jobRunID}/service/${serviceName}/log`);
    }

    getServiceLink(projectKey: string, workflowName: string, nodeRunID: number,
        jobRunID: number, serviceName: string): Observable<CDNLogLink> {
        return this._http.get<CDNLogLink>(`/project/${projectKey}/workflows/${workflowName}/nodes/${nodeRunID}/job/${jobRunID}/service/${serviceName}/link`);
    }

    getNodeJobRunInfo(projectKey: string, workflowName: string, runNumber: number,
        nodeRunID: number, nodeJobRunID: number): Observable<Array<SpawnInfo>> {
        return this._http.get<Array<SpawnInfo>>(`/project/${projectKey}/workflows/${workflowName}/runs/${runNumber}/nodes/${nodeRunID}/job/${nodeJobRunID}/info`);
    }

    retentionPolicyDryRun(workflow: Workflow): Observable<WorkflowRetentoinDryRunResponse> {
        return this._http.post<WorkflowRetentoinDryRunResponse>(`/project/${workflow.project_key}/workflows/${workflow.name}/retention/dryrun`,
            { retention_policy: workflow.retention_policy });
    }

    retentionPolicySuggestion(workflow: Workflow) {
        return this._http.get<Array<string>>(`/project/${workflow.project_key}/workflows/${workflow.name}/retention/suggest`);
    }
}
