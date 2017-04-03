import {Injectable} from '@angular/core';
import {List, Map} from 'immutable';
import {BehaviorSubject, Observable} from 'rxjs/Rx';
import {Project} from '../../model/project.model';
import {ProjectService} from './project.service';
import {Variable} from '../../model/variable.model';
import {GroupPermission} from '../../model/group.model';
import {Environment} from '../../model/environment.model';
import {Notification} from '../../model/notification.model';

@Injectable()
export class ProjectStore {

    // List of all project. Use by Navbar
    private _projectNav: BehaviorSubject<List<Project>> = new BehaviorSubject(List([]));

    // List of all project + dependencies:  List of variables, List of Env, List of App, List of Pipeline.
    private _projectCache: BehaviorSubject<Map<string, Project>> = new BehaviorSubject(Map<string, Project>());

    constructor(private _projectService: ProjectService) {

    }

    /**
     * Get a Project Observable
     * @returns {Observable<List<Project>>}
     */
    getProjectsList(): Observable<List<Project>> {
        // If Store not empty, get from it
        if (this._projectNav.getValue().size === 0) {
            // Get from localstorage
            let localProjects: Array<Project> = JSON.parse(localStorage.getItem('CDS-PROJECT-LIST'));
            this._projectNav.next(this._projectNav.getValue().push(...localProjects));

            // Get from API
            this._projectService.getProjects().subscribe(res => {
                localStorage.setItem('CDS-PROJECT-LIST', JSON.stringify(res));
                this._projectNav.next(List(res));
            });
        }
        return new Observable<List<Project>>(fn => this._projectNav.subscribe(fn));

    }

    /**
     * Use by router to preload project
     * @param key
     * @returns {Observable<Project>}
     */
    getProjectResolver(key: string): Observable<Project> {
        let store = this._projectCache.getValue();
        if (store.size === 0 || !store.get(key)) {
            return this._projectService.getProject(key).map( res => {
                this._projectCache.next(store.set(key, res));
                return res;
            });
        } else {
            return Observable.of(store.get(key));
        }
    }

    /**
     * Get one specific project
     * @param key Project unique key
     * @returns {Project}
     */
    getProjects(key: string): Observable<Map<string, Project>> {
        // If Store contain the project, get IT
        let projects = this._projectCache.getValue();

        if (!projects.get(key)) {
            // Else get it from API
            this._projectService.getProject(key).subscribe(res => {
                this._projectCache.next(projects.set(key, res));
            }, err => {
                this._projectCache.error(err);
            });
        }
        return new Observable<Map<string, Project>>(fn => this._projectCache.subscribe(fn));
    }

    /**
     * Create a new Project
     * @param project Project to create
     * @returns {Project}
     */
    createProject(project: Project): Observable<Project> {
        return this._projectService.addProject(project).map(res => {
            let projects = this._projectNav.getValue();
            this._projectNav.next(projects.push(project));
            return res;
        });
    }

    /**
     * Update a project
     * @param project Project to Update
     * @returns {Project}
     */
    updateProject(project: Project): Observable<Project> {
        return this._projectService.updateProject(project).map(res => {
            // update store for navigation
            let projects = this._projectNav.getValue();
            let index = projects.findIndex(prj => prj.key === res.key);
            if (index >= 0) {
                this._projectNav.next(projects.remove(index).insert(index, res));
            } else {
                this._projectNav.next(projects.push(res));
            }


            // update project cache
            let cache = this._projectCache.getValue();
            if (cache.get(res.key)) {
                let pToUpdate = cache.get(res.key);
                pToUpdate.last_modified = res.last_modified;
                pToUpdate.name = res.name;
                this._projectCache.next(cache.set(res.key, pToUpdate));
            }
            return res;
        });
    }

    /**
     *
     * @param key
     * @param applications
     */
    updateApplications(key: string, project: Project): void {
        let cache = this._projectCache.getValue();
        let projectToUpdate = cache.get(key);
        if (projectToUpdate) {
            projectToUpdate.applications = project.applications;
            projectToUpdate.last_modified = project.last_modified;
            this._projectCache.next(cache.set(key, projectToUpdate));
        }
    }

    /**
     * Update application name in project
     * @param key Project unique key
     * @param oldName old name
     * @param newName the new name of the application
     */
    updateApplicationName(key: string, oldName: string, newName: string) {
        let cache = this._projectCache.getValue();
        let projectToUpdate = cache.get(key);
        if (projectToUpdate) {
            let index: number = projectToUpdate.applications.findIndex(app => app.name === oldName);
            if (index === -1) {
                return;
            }
            let application = projectToUpdate.applications[index];
            application.name = newName;
            projectToUpdate.applications[index] = application;
            this._projectCache.next(cache.set(key, projectToUpdate));
        }
    }

    /**
     * Connect a repo manager to the given project.
     * @param key Project unique key
     * @param repoName Repo manager to connect
     * @returns {Observable<any>}
     */
    connectRepoManager(key: string, repoName: string): Observable<any> {
        return this._projectService.connectRepoManager(key, repoName).map( res => {
            let cache = this._projectCache.getValue();
            if (cache.get(key)) {
                let pToUpdate = cache.get(key);
                pToUpdate.last_modified = Number(res.last_modified);
                this._projectCache.next(cache.set(key, pToUpdate));
            }
            return res;
        });
    }

    /**
     * Send verification code to connect repomanager on project
     * @param key Project unique key
     * @param repoName Repository name
     * @param token Oauth token
     * @param verifier Verification code
     * @returns {Observable<Project>}
     */
    verificationCallBackRepoManager(key: string, repoName: string, token: string, verifier: string): Observable<Project> {
        return this._projectService.callback(key, repoName, token, verifier).map( res => {
            let cache = this._projectCache.getValue();
            let projectToUpdate = cache.get(key);
            if (projectToUpdate) {
                projectToUpdate.last_modified = res.last_modified;
                projectToUpdate.repositories_manager = res.repositories_manager;
                this._projectCache.next(cache.set(key, projectToUpdate));
            }
            return res;
        });
    }

    /**
     * Disconnect a repo manager from the given project.
     * @param key Project unique key
     * @param repoName Repo manager to disconnect
     * @returns {Observable<Project>}
     */
    disconnectRepoManager(key: string, repoName: string): Observable<Project> {
        return this._projectService.disconnectRepoManager(key, repoName).map( res => {
            let cache = this._projectCache.getValue();
            let pToUpdate = cache.get(key);
            if (pToUpdate) {
                pToUpdate.last_modified = res.last_modified;
                if (pToUpdate.repositories_manager) {
                    let indexRepo: number = pToUpdate.repositories_manager.findIndex(r => r.name === repoName);
                    if (indexRepo >= 0) {
                        pToUpdate.repositories_manager.splice(indexRepo, 1);
                        this._projectCache.next(cache.set(key, pToUpdate));
                    }
                }
            }
            return res;
        });
    }

    /**
     * Delete the given project
     * @param key Project unique key
     * @returns {Observable<boolean>}
     */
    deleteProject(key: string): Observable<boolean> {
        return this._projectService.deleteProject(key).map(res => {
            let projects = this._projectNav.getValue();
            let index = projects.findIndex(prj => prj.key === key);
            this._projectNav.next(projects.delete(index));

            let cache = this._projectCache.getValue();
            this._projectCache.next(cache.delete(key));

            return res;
        });
    }

    /**
     * Add a variable for the given project
     * @param key Project unique key
     * @param variable Variable to add
     * @returns {Observable<Project>}
     */
    addProjectVariable(key: string, variable: Variable): Observable<Project> {
        return this._projectService.addVariable(key, variable).map(res => {
            return this.refreshProjectVariableCache(key, res);
        });
    }

    /**
     * Update a variable for the given project
     * @param key Project unique key
     * @param variable Variable to update
     * @returns {Observable<Project>}
     */
    updateProjectVariable(key: string, variable: Variable): Observable<Project> {
        return this._projectService.updateVariable(key, variable).map(res => {
            return this.refreshProjectVariableCache(key, res);
        });
    }

    /**
     * Delete a variable for the given project
     * @param key Project unique key
     * @param variable Variable to delete
     * @returns {Observable<Project>}
     */
    deleteProjectVariable(key: string, variable: Variable): Observable<Project> {
        return this._projectService.removeVariable(key, variable.name).map(res => {
            return this.refreshProjectVariableCache(key, res);
        });
    }

    /**
     * Refresh permissions in cache for the current project
     * @param key Project unique key
     * @param project Project updated
     * @returns {Project}
     */
    refreshProjectVariableCache(key: string, project: Project): Project {
        let cache = this._projectCache.getValue();
        let projectUpdate = cache.get(key);
        if (projectUpdate) {
            projectUpdate.last_modified = project.last_modified;
            projectUpdate.variables = project.variables;
            this._projectCache.next(cache.set(key, projectUpdate));
            return projectUpdate;
        }
        return project;
    }

    /**
     * Add a group permission
     * @param key Project unique key
     * @param gp Permission to add
     * @returns {Observable<Project>}
     */
    addProjectPermission(key: string, gp: GroupPermission): Observable<Project> {
        return this._projectService.addPermission(key, gp).map(res => {
            return this.refreshProjectPermissionCache(key, res);
        });
    }

    /**
     * Update a group permission
     * @param key Project unique key
     * @param gp Permission to update
     * @returns {Observable<Project>}
     */
    updateProjectPermission(key: string, gp: GroupPermission): Observable<Project> {
        gp.permission = Number(gp.permission);
        return this._projectService.updatePermission(key, gp).map(res => {
            return this.refreshProjectPermissionCache(key, res);
        });
    }

    /**
     * Remove a group permission
     * @param key Project unique key
     * @param gp Permission to delete
     * @returns {Observable<Project>}
     */
    removeProjectPermission(key: string, gp: GroupPermission): Observable<Project> {
        return this._projectService.removePermission(key, gp).map(res => {
            return this.refreshProjectPermissionCache(key, res);
        });
    }

    /**
     * Refresh permissions in cache for the current project
     * @param key Project unique key
     * @param project Project updated
     * @returns {Project}
     */
    refreshProjectPermissionCache(key: string, project: Project): Project {
        let cache = this._projectCache.getValue();
        let projectUpdate = cache.get(key);
        if (projectUpdate) {
            projectUpdate.last_modified = project.last_modified;
            projectUpdate.groups = project.groups;
            this._projectCache.next(cache.set(key, projectUpdate));
            return projectUpdate;
        }
        return project;
    }

    /**
     * Add an environment
     * @param key Project unique key
     * @param environment Environment to add
     */
    addProjectEnvironment(key: string, environment: Environment) {
        return this._projectService.addEnvironment(key, environment).map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        });
    }

    /**
     * Rename an environment
     * @param key Project unique key
     * @param environment Environment to rename
     */
    renameProjectEnvironment(key: string, oldName: string, environment: Environment) {
        return this._projectService.renameEnvironment(key, oldName, environment).map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        });
    }

    /**
     * Remove an environment
     * @param key Project unique key
     * @param environment Environment to rename
     */
    deleteProjectEnvironment(key: string, environment: Environment) {
        return this._projectService.removeEnvironment(key, environment).map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        });
    }

    /**
     * Add a variable in an environment
     * @param key Project unique key
     * @param envName Environment Name
     * @param v variable to add
     * @returns {Observable<Project>}
     */
    addEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._projectService.addEnvironmentVariable(key, envName, v).map(res => {
           return this.refreshProjectEnvironmentCache(key, res);
        });
    }

    /**
     * Update a variable in an environment
     * @param key Project unique key
     * @param envName Environment Name
     * @param v variable to update
     * @returns {Observable<Project>}
     */
    updateEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._projectService.updateEnvironmentVariable(key, envName, v).map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        });
    }

    /**
     * Remove a variable from an environment
     * @param key Project unique key
     * @param envName Environment Name
     * @param v variable to remove
     * @returns {Observable<Project>}
     */
    removeEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._projectService.removeEnvironmentVariable(key, envName, v).map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        });
    }

    /**
     * Refresh environments in cache for the current project
     * @param key Project unique key
     * @param project Project updated
     * @returns {Project}
     */
    refreshProjectEnvironmentCache(key: string, project: Project): Project {
        let cache = this._projectCache.getValue();
        let projectUpdate = cache.get(key);
        if (projectUpdate) {
            projectUpdate.last_modified = project.last_modified;
            projectUpdate.environments = project.environments;
            this._projectCache.next(cache.set(key, projectUpdate));
            return projectUpdate;
        }
        return project;
    }
}

