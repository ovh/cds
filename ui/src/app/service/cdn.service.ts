import { Injectable } from "@angular/core";
import { HttpClient, HttpHeaders, HttpParams } from "@angular/common/http";
import { catchError, map, Observable, of } from "rxjs";
import { CDNLine, CDNLinesResponse, CDNLogLink, CDNLogLinks, CDNLogsLines } from "app/model/cdn.model";

@Injectable()
export class CDNService {
	constructor(
		private _http: HttpClient
	) { }

	getLogsLinesCount(links: CDNLogLinks, item_type: string): Observable<Array<CDNLogsLines>> {
		let params = new HttpParams();
		links.datas.map(l => l.api_ref).forEach(ref => {
			params = params.append('apiRefHash', ref);
		});
		return this._http.get<Array<CDNLogsLines>>(`./cdscdn/item/${item_type}/lines`, { params });
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
}