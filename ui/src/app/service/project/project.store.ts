
import { Injectable } from '@angular/core';
import { List, Map } from 'immutable';
import { BehaviorSubject, Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { LoadOpts, Project } from '../../model/project.model';
import { NavbarService } from '../navbar/navbar.service';
import { ProjectService } from './project.service';


@Injectable()
export class ProjectStore {
    private WORKFLOW_VIEW_MODE = 'CDS-WORKFLOW-VIEW-MODE';

    // List of all project + dependencies:  List of variables, List of Env, List of App, List of Pipeline.
    private _projectCache: BehaviorSubject<Map<string, Project>> = new BehaviorSubject(Map<string, Project>());
    // List of all project. Use by Navbar
    private _projectNav: BehaviorSubject<List<Project>> = new BehaviorSubject(null);

    constructor(
        private _projectService: ProjectService,
        private _navbarService: NavbarService
    ) {

    }

    getProjectsList(): Observable<List<Project>> {
        // If Store not empty, get from it
        if (!this._projectNav.getValue() || this._projectNav.getValue().size === 0) {
            // Get from API
            this._projectService.getProjects().subscribe(res => {
                this._projectNav.next(List(res));
            });
        }
        return new Observable<List<Project>>(fn => this._projectNav.subscribe(fn));
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
     * @param key Project unique key you want to fetch
     * @returns {Project}
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

    /**
     * Update a project favorite
     * @param projectKey Project key to Update
     * @returns {Project}
     */
    updateFavorite(projectKey: string): Observable<Project> {
        return this._projectService.updateFavorite(projectKey).pipe(map(() => {
            // update project cache
            let cache = this._projectCache.getValue();
            let project = cache.get(projectKey);
            if (project) {
                project.favorite = !project.favorite;
                this._projectCache.next(cache.set(projectKey, project));
            }
            this._navbarService.getData();
            return project;
        }));
    }
}
