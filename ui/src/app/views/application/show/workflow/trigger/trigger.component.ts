import {Component, Input} from '@angular/core';
import {Trigger} from '../../../../../model/trigger.model';
import {Project} from '../../../../../model/project.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ApplicationPipeline} from '../../../../../model/application.model';
import {Prerequisite} from '../../../../../model/prerequisite.model';
import {PrerequisiteEvent} from '../../../../../shared/prerequisites/prerequisite.event.model';
import {cloneDeep} from 'lodash';
import {Parameter} from '../../../../../model/parameter.model';

@Component({
    selector: 'app-application-trigger',
    templateUrl: './trigger.html',
    styleUrls: ['./trigger.scss']
})
export class ApplicationTriggerComponent {

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

    refPrerequisites: Array<Prerequisite>;

    constructor(private _appStore: ApplicationStore) {
        this.refPrerequisites = new Array<Prerequisite>();
        this.refPrerequisites.push(this.getGitPrerequisite());
    }

    /**
     * Refresh available pipeline for the selected application.
     */
    updatePipelineList(): void {
        this._appStore.getApplications(this.project.key, this.trigger.dest_application.name).subscribe(apps => {
            let appKey = this.project.key + '-' + this.trigger.dest_application.name;
            if (apps.get(appKey)) {
                this.appPipelines = apps.get(appKey).pipelines;
            }
        });
    }

    /**
     * Update selected dest pipeline + list of prerequisite
     */
    updateDestPipeline(): void {
        this.selectedDestPipeline = this.appPipelines.filter(p => p.pipeline.name === this.trigger.dest_pipeline.name)[0].pipeline;
        this.refPrerequisites = new Array<Prerequisite>();
        this.refPrerequisites.push(this.getGitPrerequisite());
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
    getGitPrerequisite(): Prerequisite {
        let p = new Prerequisite();
        p.parameter = 'git.branch';
        p.expected_value = '';
        return p;
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
