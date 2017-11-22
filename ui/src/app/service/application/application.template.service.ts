import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Template} from '../../model/template.model';
import {HttpClient} from '@angular/common/http';

@Injectable()
export class ApplicationTemplateService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of template
     * @returns {Observable<Template>}
     */
    getTemplates(): Observable<Array<Template>> {
        return this._http.get<Array<Template>>('/template');
    }
}
