import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {HttpClient} from '@angular/common/http';
import {WorkflowHookModel} from '../../model/workflow.hook.model';

@Injectable()
export class HookService {

    private models: Array<WorkflowHookModel>;

    constructor(private _http: HttpClient) {
    }

    getHookModel(): Observable<Array<WorkflowHookModel>> {
        if (!this.models) {
            return this._http.get<Array<WorkflowHookModel>>('/workflow/hook/model').map(ms => {
                this.models = <Array<WorkflowHookModel>>ms;
                return ms;
            });
        }
        return Observable.of(this.models);
    }
}
