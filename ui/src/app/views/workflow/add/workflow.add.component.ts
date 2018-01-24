import {Component} from '@angular/core';
import {Workflow} from '../../../model/workflow.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {Application} from '../../../model/application.model';
import {Pipeline} from '../../../model/pipeline.model';
import {Environment} from '../../../model/environment.model';
import {ApplicationStore} from '../../../service/application/application.store';
import {ProjectStore} from '../../../service/project/project.store';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {first, finalize} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-add',
    templateUrl: './workflow.add.html',
    styleUrls: ['./workflow.add.scss']
})
export class WorkflowAddComponent {

    workflow: Workflow;
    project: Project;

    loading = false;
    currentStep = 0;

    // Pipeline section
    set createNewPipeline(data: boolean) {
      this._createNewPipeline = data;
      if (data) {
        this.newPipeline = new Pipeline();
        this.workflow.root.pipeline_id = null;
      }
    }
    get createNewPipeline() {
      return this._createNewPipeline;
    }
    errorPipelineNamePattern = false;
    loadingCreatePipeline = false;
    newPipeline: Pipeline = new Pipeline();
    pipelineSection: 'pipeline'|'application'|'environment' = 'pipeline';
    _createNewPipeline = false;

    // Application details
    set createNewApplication(data: boolean) {
      this._createNewApplication = data;
      if (data) {
        this.newApplication = new Application();
        this.workflow.root.context.application_id = null;
      }
    }
    get createNewApplication() {
      return this._createNewApplication;
    }
    errorApplicationNamePattern = false;
    loadingCreateApplication = false;
    newApplication: Application = new Application();
    _createNewApplication = false;

    // Environment details
    set createNewEnvironment(data: boolean) {
      this._createNewEnvironment = data;
      if (data) {
        this.newEnvironment = new Environment();
        this.workflow.root.context.environment_id = null;
      }
    }
    get createNewEnvironment() {
      return this._createNewEnvironment;
    }
    loadingCreateEnvironment = false;
    newEnvironment: Environment = new Environment();
    _createNewEnvironment = false;

    constructor(private _activatedRoute: ActivatedRoute,
                private _router: Router, private _workflowStore: WorkflowStore,
                private _translate: TranslateService, private _toast: ToastService,
                private _pipStore: PipelineStore, private _appStore: ApplicationStore,
                private _projectStore: ProjectStore) {
        this.workflow = new Workflow();

        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
            if (!this.project.pipeline_names || !this.project.pipeline_names.length) {
              this.createNewPipeline = true;
            }
            if (!this.project.application_names || !this.project.application_names.length) {
              this.createNewApplication = true;
            }
        });
    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], {queryParams: {tab: 'workflows'}});
    }

    createWorkflow(): void {
        this.loading = true;
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

    createPipeline(): void {
        if (!Pipeline.checkName(this.newPipeline.name)) {
          this.errorPipelineNamePattern = true;
          return;
        }

        this.loadingCreatePipeline = true;
        this.newPipeline.type = 'deployment';
        this._pipStore.createPipeline(this.project.key, this.newPipeline)
            .pipe(finalize(() => this.loadingCreatePipeline = false))
            .subscribe((pip) => {
                this._toast.success('', this._translate.instant('pipeline_added'));
                this.workflow.root.pipeline_id = pip.id;
                this.pipelineSection = 'application';
            });
    }

    selectOrCreatePipeline() {
      if (this.createNewPipeline) {
        return this.createPipeline();
      }
      this.pipelineSection = 'application';
    }

    createApplication(): void {
      if (!Application.checkName(this.newApplication.name)) {
        this.errorApplicationNamePattern = true;
        return;
      }

      this.loadingCreateApplication = true;
      this._appStore.createApplication(this.project.key, this.newApplication)
          .pipe(finalize(() => this.loadingCreateApplication = false))
          .subscribe((app) => {
              this._toast.success('', this._translate.instant('application_created'));
              this.workflow.root.context.application_id = app.id;
              this.pipelineSection = 'environment';
          });
    }

    selectOrCreateApplication() {
      if (this.createNewApplication && this.newApplication.name) {
        return this.createApplication();
      }
      this.pipelineSection = 'environment';
    }

    createEnvironment(): void {
      this.loadingCreateEnvironment = true;
      this._projectStore.addProjectEnvironment(this.project.key, this.newEnvironment)
          .pipe(finalize(() => this.loadingCreateEnvironment = false))
          .subscribe((proj) => {
              this._toast.success('', this._translate.instant('environment_created'));
              this.workflow.root.context.environment_id = proj.environments.find((env) => env.name === this.newEnvironment.name).id;
              this.createWorkflow();
          });
    }

    selectOrCreateEnvironment() {
      if (this.createNewEnvironment && this.newEnvironment.name) {
        return this.createEnvironment();
      }
      this.createWorkflow();
    }

    goToNextStep(stepNum: number): void {
      if (stepNum != null) {
        this.currentStep = stepNum;
      } else {
        this.currentStep++;
      }
    }
}
