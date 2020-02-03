import { ActionWarning } from './action.model';
import { Job } from './job.model';
import { WorkflowNodeConditions } from './workflow.model';
import { WorkflowNodeJobRun } from './workflow.run.model';

export class Stage {
  id: number;
  name: string;
  status: string;
  build_order: number;
  enabled: boolean;
  jobs: Array<Job>;
  run_jobs: Array<WorkflowNodeJobRun>;
  conditions: WorkflowNodeConditions;
  last_modified: number;
  warnings: Array<ActionWarning>;
  // UI params
  hasChanged: boolean;
  edit: boolean;
  ref: number;

  constructor() {
    this.ref = new Date().getTime();
    this.conditions = new WorkflowNodeConditions();
  }
}
