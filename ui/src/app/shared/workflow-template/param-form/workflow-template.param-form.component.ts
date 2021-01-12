import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnDestroy,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { FormControl } from '@angular/forms';
import { Project } from 'app/model/project.model';
import {
    ParamData,
    WorkflowTemplate,
    WorkflowTemplateApplyResult,
    WorkflowTemplateInstance
} from 'app/model/workflow-template.model';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-template-param-form',
    templateUrl: './workflow-template.param-form.html',
    styleUrls: ['./workflow-template.param-form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowTemplateParamFormComponent implements OnInit, OnDestroy {
    @ViewChild('codemirror') codemirror: any;

    _project: Project;
    @Input() set project(data: Project) {
        this._project = data;
        this.initProject();
    }
    get project() {
        return this._project;
    }
    @Input() workflowTemplate: WorkflowTemplate;
    @Input() workflowTemplateInstance: WorkflowTemplateInstance;
    @Input() parameters: ParamData;
    @Output() paramChange = new EventEmitter<ParamData>();

    vcsNames: Array<string>;
    sshKeys: Array<string>;
    pgpKeys: Array<string>;
    parameterValues: any;
    loading: boolean;
    result: WorkflowTemplateApplyResult;
    codeMirrorConfig: any;
    themeSubscription: Subscription;

    constructor(
        private _repoManagerService: RepoManagerService,
        private _cd: ChangeDetectorRef,
        private _theme: ThemeStore
    ) {
        this.codeMirrorConfig = this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            lineNumbers: true,
        };

        this.themeSubscription = this._theme.get()
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(t => {
                this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
                if (this.codemirror && this.codemirror.instance) {
                    this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
                }
            });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.initProject();
        this.changeParam();
    }

    initProject() {
        if (this.project && this.project.vcs_servers) {
            this.vcsNames = this.project.vcs_servers.map(vcs => vcs.name);
        }

        if (this.project && this.project.keys) {
            this.sshKeys = this.project.keys.filter(k => k.type === 'ssh').map(k => k.name);
            this.pgpKeys = this.project.keys.filter(k => k.type === 'pgp').map(k => k.name);
        }

        this.parameterValues = {};
        if (this.workflowTemplate && this.workflowTemplate.parameters) {
            this.workflowTemplate.parameters.forEach(parameter => {
                if (parameter.type === 'boolean') {
                    this.parameterValues[parameter.key] = false;
                }
            });
        }

        this.fillFormWithInstanceData();
    }

    fetchRepos(parameterKey: string, repoMan: string): void {
        this._repoManagerService.getRepositories(this.project.key, repoMan, false).subscribe(rs => {
            let repoNames = rs.map(r => r.fullname);

            this.parameterValues[parameterKey + '-repositories'] = repoNames;

            if (this.workflowTemplateInstance && this.workflowTemplateInstance.request.parameters[parameterKey]) {
                let v = this.workflowTemplateInstance.request.parameters[parameterKey];
                let s = v.split('/');
                if (s.length > 1) {
                    let selectedRepo = s.splice(1, s.length - 1).join('/');
                    let existingRepo = repoNames.find(n => n === selectedRepo);
                    if (existingRepo) {
                        this.parameterValues[parameterKey + '-repository'] = existingRepo;
                        this.changeParam();
                    }
                }
            }
            this._cd.markForCheck();
        });
    }

    resyncRepos(parameterKey: string) {
        if (this.parameterValues[parameterKey]) {
            this.loading = true;
            this._repoManagerService.getRepositories(this.project.key, this.parameterValues[parameterKey], true)
                .pipe(first(), finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                }))
                .subscribe(rs => {
                    this.parameterValues[parameterKey + '-repositories'] = rs.map(r => r.fullname);
                });
        }
    }

    fillFormWithInstanceData(): void {
        if (this.workflowTemplate && this.workflowTemplateInstance) {
            this.workflowTemplate.parameters.forEach(parameter => {
                let v = (this.parameters && this.parameters[parameter.key]) ?
                    this.parameters[parameter.key] : this.workflowTemplateInstance.request.parameters[parameter.key];
                if (v) {
                    switch (parameter.type) {
                        case 'boolean':
                            this.parameterValues[parameter.key] = v === 'true';
                            break;
                        case 'repository':
                            let s = v.split('/');
                            if (s.length > 1) {
                                let existingVcs = this.vcsNames.find(vcs => vcs === s[0]);
                                if (existingVcs) {
                                    this.parameterValues[parameter.key] = existingVcs;
                                    this.fetchRepos(parameter.key, existingVcs);
                                }
                            }
                            break;
                        default:
                            this.parameterValues[parameter.key] = v;
                            break;
                    }
                }
            });
        }

        if (this.workflowTemplate) {
            this.changeParam();
        }
    }

    filterRepo(options: Array<string>, query: string): Array<string> | false {
        if (!options) {
            return false;
        }
        if (!query || query.length < 3) {
            return options.slice(0, 100);
        }
        let queryLowerCase = query.toLowerCase();
        return options.filter(name => name.toLowerCase().indexOf(queryLowerCase) !== -1);
    }

    changeParam() {
        let parameters = new ParamData();

        this.workflowTemplate.parameters.forEach(parameter => {
            switch (parameter.type) {
                case 'boolean':
                    parameters[parameter.key] = this.parameterValues[parameter.key] ? 'true' : 'false';
                    break;
                case 'repository':
                    if (this.parameterValues[parameter.key + '-repository']) {
                        parameters[parameter.key] = this.parameterValues[parameter.key] + '/' +
                            this.parameterValues[parameter.key + '-repository'];
                    }
                    break;
                default:
                    if (this.parameterValues[parameter.key]) {
                        parameters[parameter.key] = this.parameterValues[parameter.key];
                    }
                    break;
            }
        });

        this.paramChange.emit(parameters);
    }
}
