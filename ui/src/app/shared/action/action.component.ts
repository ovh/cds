import {Component, EventEmitter, Input, OnDestroy, OnInit, Output} from '@angular/core';
import {cloneDeep} from 'lodash';
import {DragulaService} from 'ng2-dragula/components/dragula.provider';
import {Action} from '../../model/action.model';
import {AllKeys} from '../../model/keys.model';
import {Parameter} from '../../model/parameter.model';
import {Pipeline} from '../../model/pipeline.model';
import {Project} from '../../model/project.model';
import {Requirement} from '../../model/requirement.model';
import {ActionStore} from '../../service/action/action.store';
import {ParameterEvent} from '../parameter/parameter.event.model';
import {RequirementEvent} from '../requirements/requirement.event.model';
import {SharedService} from '../shared.service';
import {ActionEvent} from './action.event.model';
import {StepEvent} from './step/step.event';

@Component({
    selector: 'app-action',
    templateUrl: './action.html',
    styleUrls: ['./action.scss']
})
export class ActionComponent implements OnDestroy, OnInit {
    editableAction: Action;
    steps: Array<Action> = new Array<Action>();
    publicActions: Array<Action>;

    @Input() project: Project;
    @Input() keys: AllKeys;
    @Input() pipeline: Pipeline;
    @Input() edit = false;
    @Input() suggest: Array<string>;

    @Input('action')
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
    configRequirements: {disableModel?: boolean, disableHostname?: boolean} = {};
    constructor(private sharedService: SharedService, private _actionStore: ActionStore, private dragulaService: DragulaService) {
        dragulaService.setOptions('bag-nonfinal', {
            moves: function (el, source, handle) {
                return handle.classList.contains('move');
            },
        });
        dragulaService.setOptions('bag-final', {
            moves: function (el, source, handle) {
                return handle.classList.contains('move');
            },
            direction: 'vertical'
        });
        this.dragulaService.drop.subscribe( () => {
            this.editableAction.hasChanged = true;
        });
    }

    ngOnInit() {
        this._actionStore.getActions().subscribe(mapActions => {
            this.publicActions = mapActions.toArray().filter((action) => action.name !== this.editableAction.name);
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
            case 'displayChoice':
                this.editableAction.showAddStep = true;
                break;
            case 'cancel':
                // nothing to do
                break;
            case 'add':
                let newStep = cloneDeep(event.step);
                this.steps.push(newStep);
                break;
            case 'delete':
                let index = this.steps.indexOf(event.step);
                if (index >= 0 ) {
                    this.steps.splice(index, 1);
                }
                break;
        }
    }

    /**
     * Send action event
     * @param type type of event (update/delete)
     */
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
}
