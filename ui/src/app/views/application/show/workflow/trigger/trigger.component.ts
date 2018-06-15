import {Component, Input, OnInit} from '@angular/core';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs';
import {ApplicationPipeline} from '../../../../../model/application.model';
import {Parameter} from '../../../../../model/parameter.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {Prerequisite} from '../../../../../model/prerequisite.model';
import {Project} from '../../../../../model/project.model';
import {Trigger} from '../../../../../model/trigger.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {PrerequisiteEvent} from '../../../../../shared/prerequisites/prerequisite.event.model';

@Component({
    selector: 'app-application-trigger',
    templateUrl: './trigger.html',
    styleUrls: ['./trigger.scss']
})
@AutoUnsubscribe()
export class ApplicationTriggerComponent implements OnInit {

    // Trigger to edit
    @Input() trigger: Trigger;

    // Pipeline parameters
    @Input() paramsRef: Array<Parameter>;

    // Project data
    @Input() project: Project;

    // create/edit
    @Input() mode: string;

    appPipelines: Array<ApplicationPipeline>;
    selectedDestPipeline: Pipeline;
    applicationSubscription: Subscription;

    refPrerequisites: Array<Prerequisite>;
    loading = true;

    constructor(private _appStore: ApplicationStore) {
        this.refPrerequisites = this.getGitPrerequisite();
    }

    ngOnInit() {
        if (this.mode === 'edit') {
            this.updatePipelineList();
        }
    }

    /**
     * Refresh available pipeline for the selected application.
     */
    updatePipelineList(): void {
        this.applicationSubscription = this._appStore.getApplications(this.project.key, this.trigger.dest_application.name)
            .subscribe(apps => {
                let appKey = this.project.key + '-' + this.trigger.dest_application.name;
                if (apps.get(appKey)) {
                    this.appPipelines = apps.get(appKey).pipelines;
                    if (this.mode === 'edit') {
                        this.updateDestPipeline();
                    }
                }
                this.loading = false
            }, () => this.loading = false);
    }

    /**
     * Update selected dest pipeline + list of prerequisite
     */
    updateDestPipeline(): void {
        let selectedAppPipelines = this.appPipelines.filter(p => p.pipeline.name === this.trigger.dest_pipeline.name);
        if (!selectedAppPipelines.length) {
            this.trigger.parameters = [];
            this.refPrerequisites = this.getGitPrerequisite();
            return;
        }
        this.selectedDestPipeline = selectedAppPipelines[0].pipeline;
        this.refPrerequisites = this.getGitPrerequisite();

        if (this.selectedDestPipeline.parameters) {
            this.trigger.parameters = cloneDeep(this.selectedDestPipeline.parameters);
            this.selectedDestPipeline.parameters.forEach(p => {
               let pre = new Prerequisite();
               pre.parameter = p.name;
               pre.expected_value = p.value;
               this.refPrerequisites.push(pre);
            });
        }
    }

    /**
     * Return git branch prerequisite
     * @returns {Prerequisite}
     */
    getGitPrerequisite(): Prerequisite[] {
        return ['git.branch', 'git.message', 'git.author', 'git.repository'].map((paramName) => {
          let p = new Prerequisite();
          p.parameter = paramName;
          p.expected_value = '';

          return p;
        });
    }

    /**
     * Manage action on trigger prerequisite.
     * @param event Event
     */
    prerequisiteEvent(event: PrerequisiteEvent): void {
        this.trigger.hasChanged = true;
        switch (event.type) {
            case 'add':
                if (!this.trigger.prerequisites) {
                    this.trigger.prerequisites = new Array<Prerequisite>();
                }
                let indexAdd = this.trigger.prerequisites.findIndex(p => p.parameter === event.prerequisite.parameter);
                if (indexAdd === -1) {
                    this.trigger.prerequisites.push(cloneDeep(event.prerequisite));
                }
                break;
            case 'delete':
                let indexDelete = this.trigger.prerequisites.findIndex(p => p.parameter === event.prerequisite.parameter);
                if (indexDelete > -1) {
                    this.trigger.prerequisites.splice(indexDelete, 1);
                }
                break;
        }
    }
}
