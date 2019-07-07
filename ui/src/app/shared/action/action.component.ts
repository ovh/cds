import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnDestroy,
    OnInit,
    Output
} from '@angular/core';
import { Router } from '@angular/router';
import { Action } from 'app/model/action.model';
import { AllKeys } from 'app/model/keys.model';
import { Parameter } from 'app/model/parameter.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Requirement } from 'app/model/requirement.model';
import { Stage } from 'app/model/stage.model';
import { WorkerModel } from 'app/model/worker-model.model';
import { ActionService } from 'app/service/action/action.service';
import { WorkerModelService } from 'app/service/services.module';
import { ActionEvent } from 'app/shared/action/action.event.model';
import { StepEvent } from 'app/shared/action/step/step.event';
import { ParameterEvent } from 'app/shared/parameter/parameter.event.model';
import { RequirementEvent } from 'app/shared/requirements/requirement.event.model';
import { SharedService } from 'app/shared/shared.service';
import cloneDeep from 'lodash-es/cloneDeep';
import { DragulaService } from 'ng2-dragula';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-action',
    templateUrl: './action.html',
    styleUrls: ['./action.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionComponent implements OnDestroy, OnInit {
    editableAction: Action;
    steps: Array<Action> = new Array<Action>();
    publicActions: Array<Action> = new Array<Action>();

    @Input() project: Project;
    @Input() keys: AllKeys;
    @Input() pipeline: Pipeline;
    @Input() stage: Stage;
    @Input() edit = false;
    @Input() suggest: Array<string>;

    @Input()
    set action(data: Action) {
        this.editableAction = cloneDeep(data);
        this.editableAction.showAddStep = false;
        if (!this.editableAction.requirements) {
            this.editableAction.requirements = new Array<Requirement>();
        } else {
            this.prepareEditRequirements();
        }
        this.steps = new Array<Action>();
        if (this.editableAction.actions) {
            this.steps = cloneDeep(this.editableAction.actions);
        }
    }

    @Output() actionEvent = new EventEmitter<ActionEvent>();

    collapsed = true;
    configRequirements: { disableModel?: boolean, disableHostname?: boolean } = {};
    workerModels: Array<WorkerModel>;

    constructor(
        private sharedService: SharedService,
        private _actionService: ActionService,
        private dragulaService: DragulaService,
        private _router: Router,
        private _workerModelService: WorkerModelService,
        public _cd: ChangeDetectorRef
    ) {
        dragulaService.createGroup('bag-nonfinal', {
            moves: function (el, source, handle) {
                return handle.classList.contains('move');
            },
        });
        dragulaService.createGroup('bag-final', {
            moves: function (el, source, handle) {
                return handle.classList.contains('move');
            },
            direction: 'vertical'
        });
        this.dragulaService.drop().subscribe(() => {
            this.editableAction.hasChanged = true;
        });
    }

    keyEvent(event: KeyboardEvent) {
        if (event.key === 's' && (event.ctrlKey || event.metaKey)) {
            event.preventDefault();
            setTimeout(() => this.sendActionEvent('update'));
        }
    }

    ngOnInit() {
        this._actionService.getAllForProject(this.project.key).pipe(finalize(() => this._cd.markForCheck())).subscribe(as => {
            this.publicActions = as;
        });
        this._workerModelService.getAllForProject(this.project.key).pipe(finalize(() => this._cd.markForCheck())).subscribe(wms => {
            this.workerModels = wms;
        });
    }

    ngOnDestroy() {
        this.dragulaService.destroy('bag-nonfinal');
        this.dragulaService.destroy('bag-final');
    }

    getDescriptionHeight(): number {
        return this.sharedService.getTextAreaheight(this.editableAction.description);
    }

    /**
     * Manage Requirement Event
     * @param r event
     */
    requirementEvent(r: RequirementEvent): void {
        this.editableAction.hasChanged = true;
        switch (r.type) {
            case 'add':
                if (!this.editableAction.requirements) {
                    this.editableAction.requirements = new Array<Requirement>();
                }
                let indexAdd = this.editableAction.requirements.findIndex(req => r.requirement.value === req.value);
                if (indexAdd === -1) {
                    this.editableAction.requirements = Object.assign([], this.editableAction.requirements);
                    this.editableAction.requirements.push(r.requirement);
                }
                if (r.requirement.type === 'model') {
                    this.configRequirements.disableModel = true;
                }
                if (r.requirement.type === 'hostname') {
                    this.configRequirements.disableHostname = true;
                }
                break;
            case 'delete':
                let indexDelete = this.editableAction.requirements.indexOf(r.requirement);
                if (indexDelete >= 0) {
                    this.editableAction.requirements.splice(indexDelete, 1);
                }
                if (r.requirement.type === 'model') {
                    this.configRequirements.disableModel = false;
                }
                if (r.requirement.type === 'hostname') {
                    this.configRequirements.disableHostname = false;
                }
                break;
        }
    }

    prepareEditRequirements(): void {
        this.configRequirements = {};
        this.editableAction.requirements.forEach(req => {
            if (req.type === 'model' || req.type === 'service') {
                let spaceIdx = req.value.indexOf(' ');
                if (spaceIdx > 1) {
                    let newValue = req.value.substring(0, spaceIdx);
                    let newOpts = req.value.substring(spaceIdx + 1, req.value.length);
                    req.value = newValue.trim();
                    req.opts = newOpts.replace(/\s/g, '\n');
                }
            }
            if (req.type === 'model') {
                this.configRequirements.disableModel = true;
            }
            if (req.type === 'hostname') {
                this.configRequirements.disableHostname = true;
            }
        });
    }

    parseRequirements(): void {
        // for each type 'model' and 'service', concat value with opts
        // and replace \n with space
        this.editableAction.requirements.forEach(req => {
            if ((req.type === 'model' || req.type === 'service') && req.opts) {
                let spaceIdx = req.value.indexOf(' ');
                let newValue = req.value;
                // if there is a space in name and opts not empty
                // override name with opts only
                if (spaceIdx > 1 && req.opts !== '') {
                    newValue = req.value.substring(0, spaceIdx);
                }
                let newOpts = req.opts.replace(/\n/g, ' ');
                req.value = (newValue + ' ' + newOpts).trim();
                req.opts = '';
            }
        })
    }

    /**
     * Manage Parameter Event
     * @param p event
     */
    parameterEvent(p: ParameterEvent): void {
        this.editableAction.hasChanged = true;
        switch (p.type) {
            case 'add':
                if (!this.editableAction.parameters) {
                    this.editableAction.parameters = new Array<Parameter>();
                }
                let indexAdd = this.editableAction.parameters.findIndex(param => p.parameter.name === param.name);
                if (indexAdd === -1) {
                    this.editableAction.parameters = this.editableAction.parameters.concat([p.parameter]);
                }
                break;
            case 'delete':
                let indexDelete = this.editableAction.parameters.indexOf(p.parameter);
                if (indexDelete >= 0) {
                    this.editableAction.parameters.splice(indexDelete, 1);
                    this.editableAction.parameters = this.editableAction.parameters.concat([]);
                }
                break;
        }
    }

    stepManagement(event: StepEvent): void {
        this.editableAction.hasChanged = true;
        this.editableAction.showAddStep = false;
        switch (event.type) {
            case 'expend':
                this.editableAction.showAddStep = true;
                break;
            case 'cancel':
                // nothing to do
                break;
            case 'add':
                let newStep = cloneDeep(event.step);
                newStep.enabled = true;
                this.steps.push(newStep);
                break;
            case 'delete':
                let index = this.steps.indexOf(event.step);
                if (index >= 0) {
                    this.steps.splice(index, 1);
                }
                break;
        }
    }

    sendActionEvent(type: string): void {
        // Rebuild step
        this.parseRequirements();
        this.editableAction.actions = new Array<Action>();
        if (this.steps) {
            this.steps.forEach(s => {
                this.editableAction.actions.push(s);
            });
        }

        this.actionEvent.emit(new ActionEvent(type, this.editableAction));
    }

    initActionFromJob(): void {
        this._router.navigate(['settings', 'action', 'add'], {
            queryParams: {
                from: `${this.project.key}/${this.pipeline.name}/${this.stage.id}/${this.editableAction.name}`
            }
        });
    }
}
