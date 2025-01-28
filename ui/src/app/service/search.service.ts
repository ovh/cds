import { HttpClient, HttpHeaders, HttpParams } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { SearchResponse, SearchResult } from "app/model/search.model";
import { map, Observable } from "rxjs";

@Injectable()
export class SearchService {

	constructor(
		private _http: HttpClient
	) { }

	search(query: string, offset: number, limit: number): Observable<SearchResponse> {
		const params = new HttpParams().appendAll({
			query,
			offset,
			limit
		});
		return this._http.get<Array<SearchResult>>(`/search`, { params, observe: 'response' }).pipe(map(res => {
			let headers: HttpHeaders = res.headers;
			return {
				totalCount: parseInt(headers.get('X-Total-Count'), 10),
				results: res.body as Array<SearchResult>
			};
		}));
	}
}
