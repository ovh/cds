import {Parameter} from './parameter.model';

export class Scheduler {
    id: number;
    environment_name: string;
    args: Parameter[];
    crontab: string;
    timezone: string;
    disabled: boolean;
    last_execution: SchedulerExecution;
    next_execution: SchedulerExecution;

    // UI params
    hasChanged: boolean;
    updating: boolean;
}

export class SchedulerExecution {
    id: number;
    execution_planned_date: string;
    execution_date: Date;
    executed: boolean;
    pipeline_build_version: number;
}
