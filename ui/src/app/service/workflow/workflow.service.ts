import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Operation } from 'app/model/operation.model';
import { BuildResult, CDNLogLink, ServiceLog, SpawnInfo } from 'app/model/pipeline.model';
import { RunToKeep } from 'app/model/purge.model';
import { Workflow, WorkflowPull, WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { Observable } from 'rxjs';

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

    retentionPolicyDryRun(workflow: Workflow): Observable<Array<RunToKeep>> {
        return this._http.post<Array<RunToKeep>>(`/project/${workflow.project_key}/workflows/${workflow.name}/retention/dryrun`,
            { retention_policy: workflow.retention_policy});
    }

    retentionPolicySuggestion(workflow: Workflow) {
        return this._http.get<Array<string>>(`/project/${workflow.project_key}/workflows/${workflow.name}/retention/suggest`);
    }


}
