import { Component, EventEmitter, Input, OnDestroy, Output } from '@angular/core';
import { Action } from 'app/model/action.model';
import { Group } from 'app/model/group.model';
import { AllKeys } from 'app/model/keys.model';
import { Parameter } from 'app/model/parameter.model';
import { Requirement } from 'app/model/requirement.model';
import { WorkerModel } from 'app/model/worker-model.model';
import { ActionService } from 'app/service/action/action.service';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { StepEvent } from 'app/shared/action/step/step.event';
import { ParameterEvent } from 'app/shared/parameter/parameter.event.model';
import { RequirementEvent } from 'app/shared/requirements/requirement.event.model';
import { SharedService } from 'app/shared/shared.service';
import cloneDeep from 'lodash-es/cloneDeep';
import { DragulaService } from 'ng2-dragula';

@Component({
    selector: 'app-action-form',
    templateUrl: './action.form.html',
    styleUrls: ['./action.form.scss']
})
export class ActionFormComponent implements OnDestroy {
    @Input() keys: AllKeys;
    @Input() suggest: Array<string>;
    @Input() groups: Array<Group>;
    @Input() loading: boolean;

    _action: Action;
    @Input() set action(a: Action) {
        this._action = { ...a };

        if (!this._action) {
            this._action = <Action>{ editable: true };
        }

        if (!this._action.requirements) {
            this._action.requirements = new Array<Requirement>();
        } else {
            this.prepareEditRequirements();
        }
        this.steps = new Array<Action>();
        if (this._action.actions) {
            this.steps = cloneDeep(this._action.actions);
        }

        this.refreshActions();
    }
    get action(): Action { return this._action; }

    @Output() save = new EventEmitter<Action>();
    @Output() delete = new EventEmitter();

    steps: Array<Action> = new Array<Action>();
    actions: Array<Action> = new Array<Action>();
    collapsed = true;
    configRequirements: { disableModel?: boolean, disableHostname?: boolean } = {};
    stepFormExpended: boolean;
    workerModels: Array<WorkerModel>;

    constructor(
        private sharedService: SharedService,
        private _actionService: ActionService,
        private dragulaService: DragulaService,
        private _workerModelService: WorkerModelService
    ) {
        dragulaService.createGroup('bag-nonfinal', {
            moves: (el, source, handle) => {
                return handle.classList.contains('move');
            },
        });
        dragulaService.createGroup('bag-final', {
            moves: (el, source, handle) => {
                return handle.classList.contains('move');
            },
            direction: 'vertical'
        });
        this.dragulaService.drop().subscribe(() => {
            this.action.hasChanged = true;
        });
    }

    keyEvent(event: KeyboardEvent) {
        if (event.key === 's' && (event.ctrlKey || event.metaKey)) {
            event.preventDefault();
            setTimeout(() => this.saveAction());
        }
    }

    refreshActions(): void {
        if (this.action.group_id) {
            this._actionService.getAllForGroup(this.action.group_id).subscribe(as => {
                this.actions = as.filter(a => this.action.id !== a.id);
            });
            this._workerModelService.getAllForGroup(this.action.group_id).subscribe(wms => {
                this.workerModels = wms;
            });
        }
    }

    ngOnDestroy() {
        this.dragulaService.destroy('bag-nonfinal');
        this.dragulaService.destroy('bag-final');
    }

    getDescriptionHeight(): number {
        return this.sharedService.getTextAreaheight(this.action.description);
    }

    /**
     * Manage Requirement Event
     * @param r event
     */
    requirementEvent(r: RequirementEvent): void {
        this.action.hasChanged = true;
        switch (r.type) {
            case 'add':
                if (!this.action.requirements) {
                    this.action.requirements = new Array<Requirement>();
                }
                let indexAdd = this.action.requirements.findIndex(req => r.requirement.value === req.value);
                if (indexAdd === -1) {
                    this.action.requirements = Object.assign([], this.action.requirements);
                    this.action.requirements.push(r.requirement);
                }
                if (r.requirement.type === 'model') {
                    this.configRequirements.disableModel = true;
                }
                if (r.requirement.type === 'hostname') {
                    this.configRequirements.disableHostname = true;
                }
                break;
            case 'delete':
                let indexDelete = this.action.requirements.indexOf(r.requirement);
                if (indexDelete >= 0) {
                    this.action.requirements.splice(indexDelete, 1);
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
        this.action.requirements.forEach(req => {
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
        this.action.requirements.forEach(req => {
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
        this.action.hasChanged = true;
        switch (p.type) {
            case 'add':
                if (!this.action.parameters) {
                    this.action.parameters = new Array<Parameter>();
                }
                let indexAdd = this.action.parameters.findIndex(param => p.parameter.name === param.name);
                if (indexAdd === -1) {
                    this.action.parameters = this.action.parameters.concat([p.parameter]);
                }
                break;
            case 'delete':
                let indexDelete = this.action.parameters.indexOf(p.parameter);
                if (indexDelete >= 0) {
                    this.action.parameters.splice(indexDelete, 1);
                    this.action.parameters = this.action.parameters.concat([]);
                }
                break;
        }
    }

    stepManagement(event: StepEvent): void {
        this.action.hasChanged = true;
        this.stepFormExpended = false;
        switch (event.type) {
            case 'expend':
                this.stepFormExpended = true;
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

    saveAction(): void {
        // rebuild step
        this.parseRequirements();

        this.save.emit({
            ...this.action,
            group_id: Number(this.action.group_id),
            actions: this.steps ? this.steps : []
        });
    }

    deleteAction(): void {
        this.delete.emit();
    }
}
