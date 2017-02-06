import {Component, OnDestroy, NgZone} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Application} from '../../../../model/application.model';
import {Project} from '../../../../model/project.model';
import {Pipeline, PipelineBuild} from '../../../../model/pipeline.model';
import {Subscription} from 'rxjs/Rx';
import {environment} from '../../../../../environments/environment';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {CDSWorker} from '../../../../shared/worker/worker';
import {ApplicationPipelineService} from '../../../../service/application/pipeline/application.pipeline.service';
import {Commit} from '../../../../model/repositories.model';

@Component({
    selector: 'app-pipeline-build',
    templateUrl: './pipeline.build.html',
    styleUrls: ['./pipeline.build.scss']
})
export class ApplicationPipelineBuildComponent implements OnDestroy {

    // Datas
    project: Project;
    application: Application;
    pipeline: Pipeline;
    currentBuildNumber: number;

    // Allow angular update from work started outside angular context
    zone: NgZone;

    // Worker CDS that pull data
    worker: CDSWorker;

    workerSubscription: Subscription;

    selectedTab = 'pipeline';

    // tab datas
    nbTests = 0;
    nbArtifacts = 0;


    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore,
                private _router: Router, private _appPipService: ApplicationPipelineService) {
        this.zone = new NgZone({enableLongStackTrace: false});

        // Get from pipeline resolver
        this._activatedRoute.data.subscribe(data => {
            this.pipeline = data['pipeline'];
            this.application = data['application'];
            this.project = data['project'];
        });

        let envName = this._activatedRoute.snapshot.queryParams['envName'];
        this._activatedRoute.queryParams.subscribe( q => {
            envName = q['envName'];
        });
        this._activatedRoute.params.subscribe(params => {
            let buildNumber = params['buildNumber'];

            if (buildNumber && envName) {
                this.currentBuildNumber = Number(buildNumber);
                if (this.workerSubscription) {
                    this.workerSubscription.unsubscribe();
                }
                this.worker = new CDSWorker('./assets/worker/shared/runpipeline.js', './assets/worker/web/runpipeline.js');
                this.worker.start({
                    user: this._authStore.getUser(),
                    api: environment.apiURL,
                    key: this.project.key,
                    appName: this.application.name,
                    pipName: this.pipeline.name,
                    envName: envName,
                    buildNumber: buildNumber
                });

                this.worker.response().subscribe( msg => {
                    if (msg.data) {
                        let build: PipelineBuild = JSON.parse(msg.data);
                        this.zone.run(() => {
                            if (build.artifacts) {
                                if (build.artifacts.length !== this.nbArtifacts) {
                                    this.nbArtifacts = build.artifacts.length;
                                }
                            }
                            if (build.tests) {
                                if (build.tests.total !== this.nbTests) {
                                    this.nbTests = build.tests.total;
                                }
                            }
                        });
                    }
                });
            }
        });
    }

    ngOnDestroy(): void {
        if (this.worker) {
            this.worker.updateWorker('unsubscribe', {});
        }

        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    showTab(tab: string): void {
        this.selectedTab = tab;
        this._router.navigateByUrl( '/project/' + this.project.key +
            '/application/' + this.application.name +
            '/pipeline/' + this.pipeline.name +
            '/build/' + this.currentBuildNumber +
            '?tab=' + tab);
    }
}
