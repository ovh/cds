import {Component, Input, Output, EventEmitter, OnInit} from '@angular/core';
import {WorkflowNode} from '../../../../model/workflow.model';
import {Router} from '@angular/router';
import {Project} from '../../../../model/project.model';
import {Application} from '../../../../model/application.model';
import {Pipeline} from '../../../../model/pipeline.model';
import {Environment} from '../../../../model/environment.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {ProjectStore} from '../../../../service/project/project.store';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../../shared/toast/ToastService';
import {first, finalize} from 'rxjs/operators';
import {Observable} from 'rxjs/Observable';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow-node-add-wizard',
    templateUrl: './node.wizard.html',
    styleUrls: ['./node.wizard.scss']
})
export class WorkflowNodeAddWizardComponent implements OnInit {

    @Input('project')
    set project(data: Project) {
      this._project = cloneDeep(data);
    }
    get project(): Project {
      return this._project;
    }
    @Input() hideCancel: boolean;
    @Input() hideNext: boolean;
    @Output() nodeCreated: EventEmitter<WorkflowNode> = new EventEmitter<WorkflowNode>();

    _project: Project;
    node: WorkflowNode = new WorkflowNode();
    loading = false;

    // Pipeline section
    set createNewPipeline(data: boolean) {
      this._createNewPipeline = data;
      if (data) {
        this.newPipeline = new Pipeline();
        this.node.pipeline_id = null;
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
        this.node.context.application_id = null;
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
        this.node.context.environment_id = null;
      }
    }
    get createNewEnvironment() {
      return this._createNewEnvironment;
    }
    loadingCreateEnvironment = false;
    newEnvironment: Environment = new Environment();
    _createNewEnvironment = false;

    constructor(private _router: Router,
                private _translate: TranslateService,
                private _toast: ToastService,
                private _pipStore: PipelineStore,
                private _appStore: ApplicationStore,
                private _projectStore: ProjectStore) {

    }

    ngOnInit() {
      if (!this.project.pipeline_names || !this.project.pipeline_names.length) {
        this.createNewPipeline = true;
      }
      if (!this.project.application_names || !this.project.application_names.length) {
        this.createNewApplication = true;
      }

      if (Array.isArray(this.project.application_names)) {
          let voidApp = new Application();
          voidApp.id = 0;
          voidApp.name = ' ';
          this.project.application_names.unshift(voidApp);
      }
      if (Array.isArray(this.project.environments)) {
          let voidEnv = new Environment();
          voidEnv.id = 0;
          voidEnv.name = ' ';
          this.project.environments.unshift(voidEnv);
      }
    }

    goToProject(): void {
        this._router.navigate(['/project', this.project.key], {queryParams: {tab: 'workflows'}});
    }

    createNode(): void {
        this.loading = true;
        if (this.node.pipeline_id) {
            this.node.pipeline_id = Number(this.node.pipeline_id);
        }
        if (this.node.context.application_id) {
            this.node.context.application_id = Number(this.node.context.application_id);
        }
        if (this.node.context.environment_id) {
            this.node.context.environment_id = Number(this.node.context.environment_id);
        }

        this.nodeCreated.emit(this.node);
    }

    createPipeline(): Observable<Pipeline> {
        if (!Pipeline.checkName(this.newPipeline.name)) {
          this.errorPipelineNamePattern = true;
          return Observable.of(null);
        }

        this.loadingCreatePipeline = true;
        this.newPipeline.type = 'deployment';
        return this._pipStore.createPipeline(this.project.key, this.newPipeline)
            .pipe(
              first(),
              finalize(() => this.loadingCreatePipeline = false)
            )
            .map((pip) => {
                this._toast.success('', this._translate.instant('pipeline_added'));
                this.node.pipeline_id = pip.id;
                this.pipelineSection = 'application';
                return pip
            });
    }

    selectOrCreatePipeline(): Observable<string> {
      if (this.createNewPipeline) {
        return this.createPipeline()
          .map(() => 'application');
      }
      this.pipelineSection = 'application';
      return Observable.of('application');
    }

    createApplication(): Observable<Application> {
      if (!Application.checkName(this.newApplication.name)) {
        this.errorApplicationNamePattern = true;
        return Observable.of(null);
      }

      this.loadingCreateApplication = true;
      return this._appStore.createApplication(this.project.key, this.newApplication)
          .pipe(
            first(),
            finalize(() => this.loadingCreateApplication = false)
          )
          .map((app) => {
              this._toast.success('', this._translate.instant('application_created'));
              this.node.context.application_id = app.id;
              this.pipelineSection = 'environment';
              return app;
          });
    }

    selectOrCreateApplication(): Observable<string> {
      if (this.createNewApplication && this.newApplication.name) {
        return this.createApplication()
          .map(() => 'environment');
      }
      this.pipelineSection = 'environment';
      return Observable.of('environment');
    }

    createEnvironment(): Observable<Project>  {
      this.loadingCreateEnvironment = true;
      return this._projectStore.addProjectEnvironment(this.project.key, this.newEnvironment)
          .pipe(
            first(),
            finalize(() => this.loadingCreateEnvironment = false)
          )
          .map((proj) => {
              this._toast.success('', this._translate.instant('environment_created'));
              this.node.context.environment_id = proj.environments.find((env) => env.name === this.newEnvironment.name).id;
              this.createNode();
              return proj;
          });
    }

    selectOrCreateEnvironment() {
      if (this.createNewEnvironment && this.newEnvironment.name) {
        return this.createEnvironment()
          .map(() => 'done');
      }
      this.createNode();
      return Observable.of('done');
    }

    public goToNextSection(): Observable<string> {
      switch (this.pipelineSection) {
        case 'pipeline':
          return this.selectOrCreatePipeline();
        case 'application':
          return this.selectOrCreateApplication();
        case 'environment':
          return this.selectOrCreateEnvironment();
      }
    }
}
