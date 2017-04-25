import {Prerequisite} from './prerequisite.model';
import {Job} from './job.model';
import {PipelineBuildJob} from './pipeline.model';

export class Stage {
  id: number;
  name: string;
  build_order: number;
  enabled: boolean;
  jobs: Array<Job>;
  builds: Array<PipelineBuildJob>;
  prerequisites: Array<Prerequisite>;
  last_modified: number;

  // UI params
  hasChanged: boolean;
  edit: boolean;
}
