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

declare var _: any;

@Component({
    selector: 'app-action',
    templateUrl: './action.html',
    styleUrls: ['./action.scss']
})
export class ActionComponent implements OnDestroy {

    editableAction: Action;
    publicActions: Array<Action>;
    selectedStep: Action;

    @Input() project: Project;

    @Input('action')
    set action(data: Action) {
        this.editableAction = _.cloneDeep(data);
        if (!this.editableAction.requirements) {
            this.editableAction.requirements = new Array<Requirement>();
        }
    }

    @Output() actionEvent = new EventEmitter<ActionEvent>();

    constructor(private sharedService: SharedService, private _actionStore: ActionStore, private dragulaService: DragulaService) {
        this._actionStore.getActions().subscribe(mapActions => {
            this.publicActions = mapActions.toArray();
        });

        this.dragulaService.setOptions('bag-one', {
            isContainer: function (el) {
                return el.classList.contains('dragula-container');
            },
            moves: function (el, source, handle) {
                return handle.classList.contains('move');
            }
        });
        this.dragulaService.drop.subscribe( () => {
            this.editableAction.hasChanged = true;
        });
    }

    ngOnDestroy() {
        this.dragulaService.destroy('bag-one');
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

    selectPublicAction(name: string): void {
        let index = this.publicActions.findIndex(a => a.name === name);
        if (index >= 0) {
            this.selectedStep = this.publicActions[index];
        }
    };

    addStep(): void {
        if (this.selectedStep) {
            this.editableAction.hasChanged = true;
            if (!this.editableAction.actions) {
                this.editableAction.actions = new Array<Action>();
            }
            let newStep = _.cloneDeep(this.selectedStep);
            newStep.enabled = true;
            this.editableAction.actions.push(newStep);
        }
    }

    removeStep(step): void {
        let index = this.editableAction.actions.indexOf(step);
        if (index >= 0 ) {
            this.editableAction.actions.splice(index, 1);
        }
    }

    /**
     * Send action event
     * @param type type of event (update/delete)
     */
    sendActionEvent(type: string): void {
        this.actionEvent.emit(new ActionEvent(type, this.editableAction));
    }
}
