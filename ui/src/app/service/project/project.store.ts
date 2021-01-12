
import { Injectable } from '@angular/core';
import { NavbarRecentData } from 'app/model/navbar.model';
import { LoadOpts, Project } from 'app/model/project.model';
import { List, Map } from 'immutable';
import { BehaviorSubject, Observable } from 'rxjs';
import { ProjectService } from './project.service';


@Injectable()
export class ProjectStore {
    static RECENT_PROJECTS_KEY = 'CDS-RECENT-PROJECTS';

    private WORKFLOW_VIEW_MODE = 'CDS-WORKFLOW-VIEW-MODE';

    // List of all project + dependencies:  List of variables, List of Env, List of App, List of Pipeline.
    private _projectCache: BehaviorSubject<Map<string, Project>> = new BehaviorSubject(Map<string, Project>());
    // List of all project. Use by Navbar
    private _projectNav: BehaviorSubject<List<Project>> = new BehaviorSubject(null);

    private _recentProjects: BehaviorSubject<List<NavbarRecentData>> = new BehaviorSubject(List<NavbarRecentData>());

    constructor(
        private _projectService: ProjectService,
    ) {
        this.loadRecentProjects();
    }

    loadRecentProjects(): void {
        let arrayApp = JSON.parse(localStorage.getItem(ProjectStore.RECENT_PROJECTS_KEY));
        this._recentProjects.next(List.of(...arrayApp));
    }

    getProjectsList(resync: boolean = false): Observable<List<Project>> {
        // If Store not empty, get from it
        if (resync || !this._projectNav.getValue() || this._projectNav.getValue().size === 0) {
            // Get from API
            this._projectService.getProjects().subscribe(res => {
                this._projectNav.next(List(res));
            });
        }
        return new Observable<List<Project>>(fn => this._projectNav.subscribe(fn));
    }

    /**
     * Get recent projects.
     *
     * @returns
     */
    getRecentProjects(): Observable<List<NavbarRecentData>> {
        return new Observable<List<NavbarRecentData>>(fn => this._recentProjects.subscribe(fn));
    }

    /**
     * Update recent project viewed.
     *
     * @param prj Project
     */
    updateRecentProject(prj: Project): void {
        let navbarRecentData = new NavbarRecentData();
        navbarRecentData.project_key = prj.key;
        navbarRecentData.name = prj.name;
        let currentRecentProjects: Array<NavbarRecentData> = JSON.parse(localStorage.getItem(ProjectStore.RECENT_PROJECTS_KEY));
        if (currentRecentProjects) {
            let index: number = currentRecentProjects.findIndex(p =>
                p.name === navbarRecentData.name && p.project_key === navbarRecentData.project_key
            );
            if (index >= 0) {
                currentRecentProjects.splice(index, 1);
            }
        } else {
            currentRecentProjects = new Array<NavbarRecentData>();
        }
        currentRecentProjects.splice(0, 0, navbarRecentData);
        currentRecentProjects = currentRecentProjects.splice(0, 15);
        localStorage.setItem(ProjectStore.RECENT_PROJECTS_KEY, JSON.stringify(currentRecentProjects));
        this._recentProjects.next(List(currentRecentProjects));
    }

    getWorkflowViewMode(key: string): 'blocs' | 'labels' | 'lines' {
        let o = localStorage.getItem(this.WORKFLOW_VIEW_MODE);
        if (o) {
            let j = JSON.parse(o);
            if (j[key]) {
                return j[key];
            }
        }
        return 'blocs';
    }

    setWorkflowViewMode(key: string, viewMode: 'blocs' | 'labels' | 'lines') {
        let ls = localStorage.getItem(this.WORKFLOW_VIEW_MODE);
        let j = {};
        if (ls) {
            j = JSON.parse(ls);
        }
        j[key] = viewMode;
        localStorage.setItem(this.WORKFLOW_VIEW_MODE, JSON.stringify(j));
    }

    /**
     * Get all projects
     *
     * @param key Project unique key you want to fetch
     * @returns
     */
    getProjects(key?: string, opts?: LoadOpts[]): Observable<Map<string, Project>> {
        // If Store contain the project, get IT
        let projects = this._projectCache.getValue();
        if (key && !projects.get(key)) {
            // Else get it from API
            this._projectService.getProject(key, opts).subscribe(res => {
                this._projectCache.next(projects.set(key, res));
            }, err => {
                this._projectCache.error(err);
            });
        }
        return new Observable<Map<string, Project>>(fn => this._projectCache.subscribe(fn));
    }
}
