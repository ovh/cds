import {Injectable} from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {Project} from '../../model/project.model';
import {Pipeline, PipelineAudit} from '../../model/pipeline.model';
import {Observable} from 'rxjs/Observable';

@Injectable()
export class PipelineAuditService {

    constructor(private _http: HttpClient) {

    }

    getAudit(project: Project, pipeline: Pipeline): Observable<Array<PipelineAudit>> {
        return this._http.get<Array<PipelineAudit>>('/project/' + project.key + '/pipeline/' + pipeline.name + '/audits');
    }
}
