import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {Template} from '../../model/template.model';

@Injectable()
export class ApplicationTemplateService {

    constructor(private _http: Http) {
    }

    /**
     * Get the list of template
     * @returns {Observable<Template>}
     */
    getTemplates(): Observable<Array<Template>> {
        return this._http.get('/template').map(res => res.json());
    }
}
