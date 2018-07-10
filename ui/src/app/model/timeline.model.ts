import {Project} from './project.model';

export class TimelineFilter {
    projects: Array<ProjectFilter>;

    constructor() {
        this.projects = new Array<ProjectFilter>();
    }
}

export class ProjectFilter {
    key: string;
    workflow_names:  Array<string>;

    loading = false;
    project: Project;
    display: boolean;

    constructor() {
        this.workflow_names = new Array<string>();
    }
}
