import { ChangeDetectionStrategy, ChangeDetectorRef, Component, NgZone, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Operation, PerformAsCodeResponse } from 'app/model/operation.model';
import { Project } from 'app/model/project.model';
import { Repository } from 'app/model/repositories.model';
import { VCSStrategy } from 'app/model/vcs.model';
import { WorkflowTemplate } from 'app/model/workflow-template.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { ImportAsCodeService } from 'app/service/import-as-code/import.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkflowTemplateService } from 'app/service/workflow-template/workflow-template.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { SharedService } from 'app/shared/shared.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { CreateWorkflow, ImportWorkflow } from 'app/store/workflow.action';
import { Subscription } from 'rxjs';
import { filter, finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-add',
    templateUrl: './workflow.add.html',
    styleUrls: ['./workflow.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowAddComponent implements OnInit {
    @ViewChild('codeMirror', { static: false }) codemirror: any;

    workflow: Workflow;
    project: Project;
    creationMode = 'graphical';
    codeMirrorConfig: any;
    wfToImport = `# Example of workflow
name: myWorkflow
version: v2.0
workflow:
  myBuild:
    pipeline: build
  myTest:
    depends_on:
    - myBuild
    when:
    - success
    pipeline: test`;

    repos: Array<Repository>;
    selectedRepoManager: string;
    selectedRepo: Repository;
    selectedStrategy: VCSStrategy;
    pollingImport = false;
    pollingResponse: Operation;
    webworkerSub: Subscription;
    asCodeResult: PerformAsCodeResponse;
    projectSubscription: Subscription;
    templates: Array<WorkflowTemplate>;
    selectedTemplatePath: string;
    selectedTemplate: WorkflowTemplate;
    descriptionRows: number;
    updated = false;
    loading = false;
    loadingRepo = false;
    currentStep = 0;
    duplicateWorkflowName = false;
    fileTooLarge = false;
    themeSubscription: Subscription;

    constructor(
        private _store: Store,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _import: ImportAsCodeService,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _repoManagerService: RepoManagerService,
        private _workflowTemplateService: WorkflowTemplateService,
        private _sharedService: SharedService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        this.workflow = new Workflow();
        this.selectedStrategy = new VCSStrategy();
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };
    }

    ngOnInit(): void {
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.themeSubscription = this._theme.get().pipe(finalize(() => this._cd.markForCheck())).subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });

        this.projectSubscription = this._store.select(ProjectState)
            .pipe(filter((projState) => projState.project && projState.project.key), finalize(() => this._cd.markForCheck()))
            .subscribe((projectState: ProjectStateModel) => this.project = projectState.project);

        this.fetchTemplates();
    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
    }

    createWorkflow(node: WNode): void {
        this.loading = true;
        this.workflow.workflow_data.node = node;
        this._store.dispatch(new CreateWorkflow({
            projectKey: this.project.key,
            workflow: this.workflow
        })).pipe(
            finalize(() => this.loading = false)
        ).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_added'));
            this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name]);
        });
    }

    goToNextStep(stepNum: number): void {
        if (Array.isArray(this.project.workflow_names) && this.project.workflow_names.find((w) => w.name === this.workflow.name)) {
            this.duplicateWorkflowName = true;
            return;
        }

        this.duplicateWorkflowName = false;
        if (stepNum != null) {
            this.currentStep = stepNum;
        } else {
            this.currentStep++;
        }
    }

    importWorkflow() {
        this.loading = true;
        this._store.dispatch(new ImportWorkflow({
            projectKey: this.project.key,
            wfName: null,
            workflowCode: this.wfToImport
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_added'));
                this.goToProject();
            });
    }

    fetchRepos(repoMan: string): void {
        this.loadingRepo = true;
        this._repoManagerService.getRepositories(this.project.key, repoMan, false).pipe(first(), finalize(() => {
            this.loadingRepo = false;
            this._cd.markForCheck();
        })).subscribe(rs => {
            this.repos = rs;
        });
    }

    filterRepo(options: Array<Repository>, query: string): Array<Repository> | false {
        if (!options) {
            return false;
        }
        if (!query || query.length < 3) {
            return options.slice(0, 100);
        }
        let lowerQuery = query.toLowerCase();
        return options.filter(repo => repo.fullname.toLowerCase().indexOf(lowerQuery) !== -1);
    }

    filterTemplate(options: Array<WorkflowTemplate>, query: string): Array<WorkflowTemplate> | false {
        if (!options) {
            return false;
        }
        if (!query) {
            return options.sort();
        }

        let lowerQuery = query.toLowerCase();
        return options.filter(wt => {
            return wt.name.toLowerCase().indexOf(lowerQuery) !== -1 ||
                wt.slug.toLowerCase().indexOf(lowerQuery) !== -1 ||
                wt.group.name.toLowerCase().indexOf(lowerQuery) !== -1 ||
                `${wt.group.name}/${wt.slug}`.toLowerCase().indexOf(lowerQuery) !== -1;
        }).sort();
    }

    createWorkflowFromRepo() {
        let operationRequest = new Operation();
        operationRequest.strategy = this.selectedStrategy;
        if (operationRequest.strategy.connection_type === 'https') {
            operationRequest.url = this.selectedRepo.http_url;
        } else {
            operationRequest.url = this.selectedRepo.ssh_url;
        }
        operationRequest.vcs_server = this.selectedRepoManager;
        operationRequest.repo_fullname = this.selectedRepo.fullname;
        this.loading = true;
        this._import.import(this.project.key, operationRequest).pipe(first(), finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(res => {
            this.pollingImport = true;
            this.pollingResponse = res;
            if (res.status < 2) {
                this.startOperationWorker(res.uuid);
            }
        });
    }

    startOperationWorker(uuid: string): void {
        // poll operation
        let zone = new NgZone({ enableLongStackTrace: false });
        let webworker = new CDSWebWorker('./assets/worker/web/operation.js')
        webworker.start({
            'path': '/import/' + this.project.key + '/' + uuid
        });
        this.webworkerSub = webworker.response().subscribe(ope => {
            if (ope) {
                zone.run(() => {
                    this.pollingResponse = JSON.parse(ope);
                    if (this.pollingResponse.status > 1) {
                        this.pollingImport = false;
                        webworker.stop();
                    }
                    this._cd.markForCheck();
                });
            }
        });
    }

    perform(): void {
        this.loading = true;
        this._import.create(this.project.key, this.pollingResponse.uuid).pipe(first(), finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(res => {
            this.asCodeResult = res;
        });
    }

    goToWorkflow(): void {
        this._router.navigate(['/project', this.project.key, 'workflow', this.asCodeResult.workflowName]);
    }

    fileEvent(event: { content: string, file: File }) {
        this.wfToImport = event.content;
    }

    resyncRepos() {
        if (this.selectedRepoManager) {
            this.loading = true;
            this._repoManagerService.getRepositories(this.project.key, this.selectedRepoManager, true)
                .pipe(
                    first(),
                    finalize(() => {
                        this.loading = false;
                        this._cd.markForCheck();
                    })
                )
                .subscribe(repos => this.repos = repos);
        }
    }

    fileEventIcon(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000
        if (this.fileTooLarge) {
            return;
        }
        this.workflow.icon = event.content;
    }

    fetchTemplates() {
        this._workflowTemplateService.getAll().subscribe(ts => {
            this.templates = ts;
            this._cd.markForCheck();
        });
    }

    showTemplateForm(selectedTemplatePath: string) {
        this.selectedTemplate = this.templates.find(template => template.group.name + '/' + template.slug === selectedTemplatePath);
        this.descriptionRows = this._sharedService.getTextAreaheight(this.selectedTemplate.description);
    }
}
