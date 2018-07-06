import {Component, NgZone, ViewChild} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {Subscription} from 'rxjs';
import {finalize, first} from 'rxjs/operators';
import {environment} from '../../../../environments/environment';
import {Operation, PerformAsCodeResponse} from '../../../model/operation.model';
import {Project} from '../../../model/project.model';
import {Repository} from '../../../model/repositories.model';
import {VCSStrategy} from '../../../model/vcs.model';
import {Workflow, WorkflowNode} from '../../../model/workflow.model';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {ImportAsCodeService} from '../../../service/import-as-code/import.service';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../shared/toast/ToastService';
import {CDSWebWorker} from '../../../shared/worker/web.worker';

@Component({
    selector: 'app-workflow-add',
    templateUrl: './workflow.add.html',
    styleUrls: ['./workflow.add.scss']
})
@AutoUnsubscribe()
export class WorkflowAddComponent {

    workflow: Workflow;
    project: Project;

    creationMode = 'graphical';

    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    codeMirrorConfig: any;
    wfToImport = `# Example of workflow
name: myWorkflow
version: v1.0
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

    updated = false;
    loading = false;
    loadingRepo = false;
    currentStep = 0;
    duplicateWorkflowName = false;

    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore,
                private _router: Router, private _workflowStore: WorkflowStore, private _import: ImportAsCodeService,
                private _translate: TranslateService, private _toast: ToastService, private _repoManSerivce: RepoManagerService) {
        this.workflow = new Workflow();
        this.selectedStrategy = new VCSStrategy();
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };
    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], {queryParams: {tab: 'workflows'}});
    }

    createWorkflow(node: WorkflowNode): void {
        this.loading = true;
        this.workflow.root = node;
        this._workflowStore.addWorkflow(this.project.key, this.workflow)
            .pipe(
                first(),
                finalize(() => this.loading = false)
            )
            .subscribe(() => {
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
        this._workflowStore.importWorkflow(this.project.key, this.workflow.name, this.wfToImport)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_added'));
                this.goToProject();
            });
    }

    fetchRepos(repoMan: string): void {
        this.loadingRepo = true;
        this._repoManSerivce.getRepositories(this.project.key, repoMan, false).pipe(first(), finalize(() => {
            this.loadingRepo = false
        })).subscribe(rs => {
            this.repos = rs;
        })
    }

    filterRepo(options: Array<Repository>, query: string): Array<Repository> | false {
        if (!options) {
            return false;
        }
        if (!query || query.length < 3) {
            return options.slice(0, 100);
        }
        let results = options.filter(repo => repo.fullname.toLowerCase().indexOf(query.toLowerCase()) !== -1);
        return results;
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
        let zone = new NgZone({enableLongStackTrace: false});
        let webworker = new CDSWebWorker('./assets/worker/web/import-as-code.js')
        webworker.start({
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL,
            key: this.project.key,
            uuid: uuid,
        });
        this.webworkerSub = webworker.response().subscribe(ope => {
            if (ope) {
                zone.run(() => {
                    this.pollingResponse = JSON.parse(ope);
                    if (this.pollingResponse.status > 1) {
                        this.pollingImport = false;
                        webworker.stop();
                    }
                });
            }
        });
    }

    perform(): void {
        this.loading = true;
        this._import.create(this.project.key, this.pollingResponse.uuid).pipe(first(), finalize(() => {
            this.loading = false;
        })).subscribe(res => {
            this.asCodeResult = res;
        });
    }

    goToWorkflow(): void {
        this._router.navigate(['/project', this.project.key, 'workflow', this.asCodeResult.workflowName]);
    }

    fileEvent(event: {content: string, file: File}) {
        this.wfToImport = event.content;
    }

    resyncRepos() {
      if (this.selectedRepoManager) {
          this.loading = true;
          this._repoManSerivce.getRepositories(this.project.key, this.selectedRepoManager, true)
            .pipe(
              first(),
              finalize(() => this.loading = false)
            )
            .subscribe(repos => this.repos = repos);
      }
    }
}
