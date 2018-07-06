import {Project} from './project.model';

export class TimelineFilter {
    projects: Array<ProjectFilter>;
    all_projects = true;

    constructor() {
        this.projects = new Array<ProjectFilter>();
    }
}

export class ProjectFilter {
    key: string;
    workflow_names:  Array<string>;
    all_workflows: boolean;

    loading = false;
    project: Project;
    display: boolean;

    constructor() {
        this.workflow_names = new Array<string>();
        this.all_workflows = true;
    }
}
