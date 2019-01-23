import { ActionWarning } from './action.model';
import { Job } from './job.model';
import { Prerequisite } from './prerequisite.model';
import { WorkflowNodeJobRun } from './workflow.run.model';

export class Stage {
  id: number;
  name: string;
  status: string;
  build_order: number;
  enabled: boolean;
  jobs: Array<Job>;
  run_jobs: Array<WorkflowNodeJobRun>;
  prerequisites: Array<Prerequisite>;
  last_modified: number;
  warnings: Array<ActionWarning>;
  // UI params
  hasChanged: boolean;
  edit: boolean;
}
