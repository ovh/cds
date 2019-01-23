export class HeatmapSearchCriterion {
    projects: Array<any>;
    searchCriteria: string;

    constructor(projects: Array<any>, searchCriteria: string) {
        this.projects = projects
        this.searchCriteria = searchCriteria;
    }
}
