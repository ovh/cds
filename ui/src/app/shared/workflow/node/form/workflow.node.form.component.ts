import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {Project, IdName} from '../../../../model/project.model';
import {Application} from '../../../../model/application.model';
import {Environment} from '../../../../model/environment.model';
import {WorkflowNode} from '../../../../model/workflow.model';
import {cloneDeep} from 'lodash';
import { ApplicationStore } from 'app/service/services.module';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-form',
    templateUrl: './workflow.node.form.html',
    styleUrls: ['./workflow.node.form.scss']
})
export class WorkflowNodeFormComponent implements OnInit {

    @Input() project: Project;
    @Input() node: WorkflowNode;
    @Output() nodeChange = new EventEmitter<WorkflowNode>();

    environments: Environment[];
    applications: IdName[];
    platforms: Array<IdName>;

    constructor(private _appStore: ApplicationStore) { }

    ngOnInit() {
        let voidEnv = new Environment();
        voidEnv.id = 0;
        voidEnv.name = ' ';
        this.environments = cloneDeep(this.project.environments) || [];
        this.environments.unshift(voidEnv);

        let voidApp = new Application();
        voidApp.id = 0;
        voidApp.name = ' ';
        this.applications = cloneDeep(this.project.application_names) || [];
        this.applications.unshift(voidApp);
    }

    initPlatformList(): void {
        let voidPF = new IdName();
        voidPF.id = 0;
        voidPF.name = '';
        this.platforms.unshift(voidPF);
    }

    change(): void {
        this.node.context.application_id = Number(this.node.context.application_id);
        this.node.context.environment_id = Number(this.node.context.environment_id);
        this.node.pipeline_id = Number(this.node.pipeline_id);
        this.nodeChange.emit(this.node);

        let appName = this.applications.find(k => Number(k.id) === this.node.context.application_id).name
        if (appName && appName !== ' ') {
            this._appStore.getDeploymentStrategies(this.project.key, appName).pipe(
                first(),
                finalize(
                    () => {
                        this.initPlatformList();
                    }
                )
            ).subscribe(
                data => {
                    this.platforms = [];
                    let pfNames = Object.keys(data);
                    pfNames.forEach(s => {
                        let pf = this.project.platforms.find(p => p.name === s);
                        if (pf) {
                            let idName = new IdName();
                            idName.id = pf.id;
                            idName.name = pf.name;
                            this.platforms.push(idName);
                        }
                    })
                }
            )
        } else {
            this.platforms = [];
            this.initPlatformList();
            this.node.context.project_platform = null;
            this.node.context.project_platform_id = 0;
        }
    }
}
