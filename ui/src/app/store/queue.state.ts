import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import * as actionQueue from './queue.action';

export class QueueStateModel {
    public jobs: Array<WorkflowNodeJobRun>;
}
@State<QueueStateModel>({
    name: 'queue',
    defaults: {
        jobs: [],
    }
})
@Injectable()
export class QueueState {
    constructor() { }

    @Selector()
    static jobs(state: QueueStateModel) {
        return state.jobs
    }

    @Action(actionQueue.SetJobs)
    set(ctx: StateContext<QueueStateModel>, action: actionQueue.SetJobs) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            jobs: action.payload
        });
    }

    @Action(actionQueue.AddOrUpdateJob)
    addOrUpdate(ctx: StateContext<QueueStateModel>, action: actionQueue.AddOrUpdateJob) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            jobs: state.jobs.filter(j => j.id !== action.payload.id).concat(action.payload)
        });
    }

    @Action(actionQueue.RemoveJob)
    remove(ctx: StateContext<QueueStateModel>, action: actionQueue.RemoveJob) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            jobs: state.jobs.filter(j => j.id !== action.jobID)
        });
    }

    @Action(actionQueue.SetJobUpdating)
    setUpdating(ctx: StateContext<QueueStateModel>, action: actionQueue.SetJobUpdating) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            jobs: state.jobs.map(j => j.id === action.jobID ? { ...j, updating: true } : j)
        });
    }
}
