import {Pipeline} from './pipeline.model';

export class Hook {
    id: number;
    uid: string;
    pipeline: Pipeline;
    enabled: boolean;
    link: string;
    project: string;
    repository: string;


    // ui data
    updating: boolean;
    hasChanged: boolean;
}
