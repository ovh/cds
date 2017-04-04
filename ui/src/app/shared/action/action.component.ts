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

declare var _: any;

@Component({
    selector: 'app-action',
    templateUrl: './action.html',
    styleUrls: ['./action.scss']
})
export class ActionComponent implements OnDestroy {

    editableAction: Action;
    nonFinalSteps: Array<Action> = new Array<Action>();
    finalSteps: Array<Action> = new Array<Action>();
    publicActions: Array<Action>;

    @Input() project: Project;
    @Input() edit = false;

    @Input('action')
    set action(data: Action) {
        this.editableAction = _.cloneDeep(data);
        if (!this.editableAction.requirements) {
            this.editableAction.requirements = new Array<Requirement>();
        }
        this.nonFinalSteps = new Array<Action>();
        this.finalSteps = new Array<Action>();
        if (this.editableAction.actions) {
            this.editableAction.actions.forEach(s => {
                if (s.final) {
                    this.finalSteps.push(s);
                } else {
                    this.nonFinalSteps.push(s);
                }
            });
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
        switch (event.type) {
            case 'add':
                let newStep = _.cloneDeep(event.step);
                if (newStep.final) {
                    this.finalSteps.push(newStep);
                } else {
                    this.nonFinalSteps.push(newStep);
                }
                break;
            case 'delete':
                if (event.step.final) {
                    let index = this.finalSteps.indexOf(event.step);
                    if (index >= 0 ) {
                        this.finalSteps.splice(index, 1);
                    }
                } else {
                    let index = this.nonFinalSteps.indexOf(event.step);
                    if (index >= 0 ) {
                        this.nonFinalSteps.splice(index, 1);
                    }
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
        if (this.nonFinalSteps) {
            this.nonFinalSteps.forEach(s => {
                this.editableAction.actions.push(s);
            });
        }

        if (this.finalSteps) {
            this.finalSteps.forEach(s => {
                this.editableAction.actions.push(s);
            });
        }


        this.actionEvent.emit(new ActionEvent(type, this.editableAction));
    }
}
