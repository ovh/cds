import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { ApplicationService } from 'app/service/application/application.service';
import { cloneDeep } from 'lodash';
import { finalize, first } from 'rxjs/operators';
import { Environment } from '../../../../model/environment.model';
import { IdName, Project } from '../../../../model/project.model';
import { WNode, Workflow } from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-node-form',
    templateUrl: './workflow.node.form.html',
    styleUrls: ['./workflow.node.form.scss']
})
export class WorkflowNodeFormComponent implements OnInit {

    @Input() project: Project;
    @Input() node: WNode;
    @Input() workflow: Workflow;
    @Output() nodeChange = new EventEmitter<WNode>();

    environments: Environment[];
    applications: IdName[];
    integrations: Array<IdName>;

    constructor(private _appService: ApplicationService) { }

    ngOnInit() {
        let voidEnv = new Environment();
        voidEnv.id = 0;
        voidEnv.name = ' ';
        this.environments = cloneDeep(this.project.environments) || [];
        this.environments.unshift(voidEnv);

        let voidApp = new IdName();
        voidApp.id = 0;
        voidApp.name = ' ';
        this.applications = cloneDeep(this.project.application_names) || [];
        this.applications.unshift(voidApp);
    }

    initIntegrationList(): void {
        let voidPF = new IdName();
        voidPF.id = 0;
        voidPF.name = '';
        this.integrations.unshift(voidPF);
    }

    change(): void {
        this.node.context.application_id = Number(this.node.context.application_id);
        this.node.context.environment_id = Number(this.node.context.environment_id);
        this.node.context.pipeline_id = Number(this.node.context.pipeline_id);
        this.nodeChange.emit(this.node);

        let appName = this.applications.find(k => Number(k.id) === this.node.context.application_id).name
        if (appName && appName !== ' ') {
            this._appService.getDeploymentStrategies(this.project.key, appName).pipe(
                first(),
                finalize(() => this.initIntegrationList())
            ).subscribe(
                data => {
                    this.integrations = [];
                    let pfNames = Object.keys(data);
                    pfNames.forEach(s => {
                        let pf = this.project.integrations.find(p => p.name === s);
                        if (pf) {
                            let idName = new IdName();
                            idName.id = pf.id;
                            idName.name = pf.name;
                            this.integrations.push(idName);
                        }
                    })
                }
            )
        } else {
            this.integrations = [];
            this.initIntegrationList();
            this.node.context.project_integration_id = 0;
        }
    }
}
