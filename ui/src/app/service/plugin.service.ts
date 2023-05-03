import {Injectable} from "@angular/core";
import {HttpClient} from "@angular/common/http";
import {Observable} from "rxjs";
import {Plugin} from "../model/plugin.model";

/**
 * Service to access Plugin from API.
 * Only used by ProjectStore
 */
@Injectable()
export class PluginService {

    constructor(
        private _http: HttpClient
    ) {}

    getPlugin(name: string): Observable<Plugin> {
        return this._http.get<Plugin>(`/v2/plugin/${name}`);
    }
}
