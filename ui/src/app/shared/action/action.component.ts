import {Component, Input, Output, EventEmitter, OnDestroy} from '@angular/core';
import {Action} from '../../model/action.model';
import {SharedService} from '../shared.service';
import {RequirementEvent} from '../requirements/requirement.event.model';
import {Requirement} from '../../model/requirement.model';
import {ParameterEvent} from '../parameter/parameter.event.model';
import {Parameter} from '../../model/parameter.model';
import {ActionEvent} from './action.event.model';
import {ActionStore} from '../../service/action/action.store';
import {DragulaService} from 'ng2-dragula/components/dragula.provider';
import {Project} from '../../model/project.model';
import {StepEvent} from './step/step.event';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-action',
    templateUrl: './action.html',
    styleUrls: ['./action.scss']
})
export class ActionComponent implements OnDestroy {
    editableAction: Action;
    steps: Array<Action> = new Array<Action>();
    publicActions: Array<Action>;

    @Input() project: Project;
    @Input() edit = false;
    @Input() suggest: Array<string>;

    @Input('action')
    set action(data: Action) {
        this.editableAction = cloneDeep(data);
        this.editableAction.showAddStep = false;

        if (!this.editableAction.requirements) {
            this.editableAction.requirements = new Array<Requirement>();
        } else {
            this.editableAction.requirements.map(req => {
                if (req.type === 'model' || req.type === 'service') {
                    let spaceIdx = req.value.indexOf(' ');
                    if (spaceIdx > 1) {
                        let newValue = req.value.substring(0, spaceIdx);
                        let newOpts = req.value.substring(spaceIdx + 1, req.value.length);
                        req.value = newValue;
                        req.opts = newOpts.replace(/\s/g, '\n');
                    }
                }
                return req;
            })
        }
        this.steps = new Array<Action>();
        if (this.editableAction.actions) {
            this.steps = cloneDeep(this.editableAction.actions);
        }
    }

    @Output() actionEvent = new EventEmitter<ActionEvent>();

    constructor(private sharedService: SharedService, private _actionStore: ActionStore, private dragulaService: DragulaService) {
        this._actionStore.getActions().subscribe(mapActions => {
            this.publicActions = mapActions.toArray();
        });

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
                    // for type model or service, concat opts field with value
                    if (r.requirement.type === 'model' || r.requirement.type === 'service') {
                        r.requirement.value += ' ' + r.requirement.opts.replace(/\n/g, ' ');
                    }
                    this.editableAction.requirements.push(r.requirement);
                }
                break;
            case 'delete':
                let indexDelete = this.editableAction.requirements.indexOf(r.requirement);
                if (indexDelete >= 0) {
                    this.editableAction.requirements.splice(indexDelete, 1);
                }
                break;
        }
    }

    parseRequirements(): void {
        // for each type 'model' and 'service', concat value with opts
        // and replace \n with space
        this.editableAction.requirements.map(req => {
            if (req.type === 'model' || req.type === 'service' && req.opts) {
                let spaceIdx = req.value.indexOf(' ');
                let newValue = req.value;
                // if there is a space in name and opts not empty
                // override name with opts only
                if (spaceIdx > 1 && req.opts !== '') {
                    newValue = req.value.substring(0, spaceIdx);
                }
                let newOpts = req.opts.replace(/\n/g, ' ');
                req.value = newValue + ' ' + newOpts;
                req.opts = '';
            }
            return req;
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
                    this.editableAction.parameters.push(p.parameter);
                }
                break;
            case 'delete':
                let indexDelete = this.editableAction.parameters.indexOf(p.parameter);
                if (indexDelete >= 0) {
                    this.editableAction.parameters.splice(indexDelete, 1);
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
        this.editableAction.actions = new Array<Action>();
        if (this.steps) {
            this.steps.forEach(s => {
                this.editableAction.actions.push(s);
            });
        }

        this.actionEvent.emit(new ActionEvent(type, this.editableAction));
    }
}
