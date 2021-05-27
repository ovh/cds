import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { ChangeDetectorRef, EventEmitter } from '@angular/core';
import {
    ChangeDetectionStrategy,
    Component,
    OnDestroy,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { EMPTY, Observable, Subject, Subscription, timer } from 'rxjs';
import { catchError, concatMap, debounce, finalize, tap } from 'rxjs/operators';
import { WorkflowV3, WorkflowV3ValidationResponse } from '../workflowv3.model';

@Component({
    selector: 'app-workflowv3-edit',
    templateUrl: './workflowv3-edit.html',
    styleUrls: ['./workflowv3-edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3EditComponent implements OnInit, OnDestroy {
    @ViewChild('codeMirror') codemirror: any;
    codeMirrorConfig: any;

    @Output() onChange = new EventEmitter<WorkflowV3>();

    paramsRouteSubscription: Subscription;
    workflowYaml = '';
    themeSubscription: Subscription;
    workflowYamlSubject = new Subject<string>();
    errorMessage: string;
    writing = false;
    loading = false;

    constructor(
        private _theme: ThemeStore,
        private _http: HttpClient,
        private _cd: ChangeDetectorRef,
        private _activatedRoute: ActivatedRoute
    ) {
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            tabSize: 2,
            indentWithTabs: false,
            gutters: ['CodeMirror-lint-markers']
        };
    }

    // Should be set to use @AutoUnsubscribe with AOT
    ngOnDestroy(): void {
        this.workflowYamlSubject.complete();
    }

    ngOnInit(): void {
        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });

        const params = this._activatedRoute.snapshot.params;
        const projectKey = params['key'];
        const workflowName = params['workflowName'];

        this.workflowYamlSubject
            .pipe(
                tap(() => {
                    this.writing = true;
                    this._cd.markForCheck();
                }),
                debounce(() => timer(500)),
                concatMap(data => {
                    this.loading = true;
                    this._cd.markForCheck();
                    return this.validate(projectKey, data);
                }),
                catchError(err => {
                    this.writing = false;
                    this.loading = false;
                    this.errorMessage = null;
                    this._cd.markForCheck();
                    return EMPTY;
                })
            )
            .subscribe(r => {
                this.writing = false;
                this.loading = false;
                if (!r.valid) {
                    this.errorMessage = r.error;
                    this._cd.markForCheck();
                    return;
                }
                this.errorMessage = null;
                this._cd.markForCheck();
                this.onChange.emit(r.workflow);
            });

        this.loading = true;
        this._cd.markForCheck();
        this.getWorkflow(projectKey, workflowName)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(w => {
                this.workflowYaml = w;
                this.workflowYamlChange(w);
            });
    }

    workflowYamlChange(data: string) {
        this.workflowYamlSubject.next(data);
    }

    validate(projectKey: string, workflowYaml: string): Observable<WorkflowV3ValidationResponse> {
        let headers = (new HttpHeaders()).append('Content-Type', 'application/x-yaml');
        return this._http.post<WorkflowV3ValidationResponse>(`/project/${projectKey}/workflowv3/validate`, workflowYaml, { headers });
    }

    getWorkflow(projectKey: string, workflowName: string): Observable<string> {
        let params = new HttpParams();
        params = params.append('format', 'yaml');
        params = params.append('full', 'true');
        return this._http.get<string>(`/project/${projectKey}/workflowv3/${workflowName}`, { params, responseType: <any>'text' });
    }
}
