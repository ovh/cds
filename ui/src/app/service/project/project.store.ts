
import {BehaviorSubject, Observable, of as observableOf} from 'rxjs';

import {Injectable} from '@angular/core';
import {List, Map} from 'immutable';
import {map} from 'rxjs/operators';
import {Environment} from '../../model/environment.model';
import {GroupPermission} from '../../model/group.model';
import {Label, LoadOpts, Project} from '../../model/project.model';
import {Variable} from '../../model/variable.model';
import {EnvironmentService} from '../environment/environment.service';
import {NavbarService} from '../navbar/navbar.service';
import {VariableService} from '../variable/variable.service';
import {ProjectService} from './project.service';

import {ProjectIntegration} from '../../model/integration.model';
import {Key} from '../../model/keys.model';


@Injectable()
export class ProjectStore {
    private WORKFLOW_VIEW_MODE = 'CDS-WORKFLOW-VIEW-MODE';
    // List of all project. Use by Navbar
    private _projectNav: BehaviorSubject<List<Project>> = new BehaviorSubject(null);

    // List of all project + dependencies:  List of variables, List of Env, List of App, List of Pipeline.
    private _projectCache: BehaviorSubject<Map<string, Project>> = new BehaviorSubject(Map<string, Project>());

    constructor(
        private _projectService: ProjectService,
        private _environmentService: EnvironmentService,
        private _variableService: VariableService,
        private _navbarService: NavbarService
      ) {

    }

    getWorkflowViewMode(key: string): 'blocs'|'labels'|'lines' {
        let o = localStorage.getItem(this.WORKFLOW_VIEW_MODE);
        if (o) {
            let j = JSON.parse(o);
            if (j[key]) {
                return j[key];
            }
        }
        return 'blocs';
    }

    setWorkflowViewMode(key: string, viewMode: 'blocs'|'labels'|'lines') {
        let ls = localStorage.getItem(this.WORKFLOW_VIEW_MODE);
        let j = {};
        if (ls) {
            j = JSON.parse(ls);
        }
        j[key] = viewMode;
        localStorage.setItem(this.WORKFLOW_VIEW_MODE, JSON.stringify(j));
    }

    /**
     * Get a Project Observable
     * @returns {Observable<List<Project>>}
     */
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

    /**
     * Use by router to preload project
     * @param key
     * @returns {Observable<Project>}
     */
    getProjectResolver(key: string, opts: LoadOpts[]): Observable<Project> {
        let store = this._projectCache.getValue();
        if (store.size === 0 || !store.get(key)) {
            return this.resync(key, opts);
        }

        if (Array.isArray(opts) && store.get(key)) {
            let funcs = opts.filter((opt) => store.get(key)[opt.fieldName] == null);

            if (!funcs.length) {
                return observableOf(store.get(key));
            }

            return this.resync(key, funcs);
        }
        return observableOf(store.get(key));
    }

    /**
     * Get project from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resync(key: string, opts: LoadOpts[]): Observable<Project> {
        return this._projectService.getProject(key, opts).pipe(map(res => {
            let store = this._projectCache.getValue();
            let proj = store.get(key);
            if (proj) {
                proj = Object.assign({}, proj, res);
                if (opts) {
                    opts.forEach(o => {
                       switch (o.fieldName) {
                           case 'workflow_names':
                               if (!res.workflow_names) {
                                   proj.workflow_names = [];
                               }
                               break;
                           case 'pipeline_names':
                               if (!res.pipeline_names) {
                                   proj.pipeline_names = [];
                               }
                               break;
                           case 'application_names':
                               if (!res.application_names) {
                                   proj.application_names = [];
                               }
                               break;
                           case 'environments':
                               if (!res.environments) {
                                   proj.environments = [];
                               }
                               break;
                           case 'integrations':
                               if (!res.integrations) {
                                   proj.integrations = [];
                               }
                               break;
                           case 'keys':
                               if (!res.keys) {
                                   proj.keys = [];
                               }
                               break;
                           case 'labels':
                               if (!res.labels) {
                                   proj.labels = [];
                               }
                               break;
                       }
                    });
                }
            } else {
                proj = res;
            }
            this._projectCache.next(store.set(key, proj));
            return proj;
        }));
    }

    /**
     * Use by router to preload project
     * @param key
     * @returns {Observable<Project>}
     */
    getProjectEnvironmentsResolver(key: string): Observable<Project> {
        let store = this._projectCache.getValue();
        let missingEnv = store.size === 0 || !store.get(key) || !store.get(key).environments || !store.get(key).environments.length;

        if (missingEnv) {
            return this.resyncEnvironments(key);
        } else {
            return observableOf(store.get(key));
        }
    }

    /**
     * Get project from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resyncEnvironments(key: string): Observable<Project> {
        return this._environmentService.get(key).pipe(
          map((res) => {
              let store = this._projectCache.getValue();
              let proj = store.get(key);
              proj.environments = res;
              this._projectCache.next(store.set(key, proj));
              return proj;
          }));
    }

    /**
     * Use by router to preload project
     * @param key
     * @returns {Observable<Project>}
     */
    getProjectApplicationsResolver(key: string): Observable<Project> {
        let store = this._projectCache.getValue();
        let missingApps = store.size === 0 || !store.get(key) || !store.get(key).applications || !store.get(key).applications.length;

        if (missingApps) {
            return this.resyncApplications(key);
        } else {
            return observableOf(store.get(key));
        }
    }

    /**
     * Get project applications from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resyncApplications(key: string): Observable<Project> {
        return this._projectService.getApplications(key).pipe(
          map((res) => {
              let store = this._projectCache.getValue();
              let proj = store.get(key);
              proj.applications = res;
              this._projectCache.next(store.set(key, proj));
              return proj;
          }));
    }

    getProjectKeysResolver(key: string): Observable<Project> {
        let store = this._projectCache.getValue();
        let missingKeys = store.size === 0 || !store.get(key) || !store.get(key).keys || !store.get(key).keys.length;
        if (missingKeys) {
            return this.resyncKeys(key);
        } else {
            return observableOf(store.get(key));
        }
    }

    /**
     * Use by router to preload project
     * @param key
     * @returns {Observable<Project>}
     */
    getProjectVariablesResolver(key: string): Observable<Project> {
        let store = this._projectCache.getValue();
        let missingEnv = store.size === 0 || !store.get(key) || !store.get(key).variables || !store.get(key).variables.length;

        if (missingEnv) {
            return this.resyncVariables(key);
        } else {
            return observableOf(store.get(key));
        }
    }

    resyncKeys(key: string): Observable<Project> {
        return this._projectService.getKeys(key).pipe(
            map((res) => {
                let store = this._projectCache.getValue();
                let proj = store.get(key);
                proj.keys = res;
                this._projectCache.next(store.set(key, proj));
                return proj;
            }));
    }

    /**
     * Get project from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resyncVariables(key: string): Observable<Project> {
        return this._variableService.get(key).pipe(
          map((res) => {
              let store = this._projectCache.getValue();
              let proj = store.get(key);
              proj.variables = res;
              this._projectCache.next(store.set(key, proj));
              return proj;
          }));
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
     * Create a new Project
     * @param project Project to create
     * @returns {Project}
     */
    createProject(project: Project): Observable<Project> {
        return this._projectService.addProject(project).pipe(map(res => {
            let projects = this._projectNav.getValue();
            if (!projects) {
                projects = List();
            }
            this._projectNav.next(projects.push(project));
            return res;
        }));
    }

    /**
     * Update a project
     * @param project Project to Update
     * @returns {Project}
     */
    updateProject(project: Project): Observable<Project> {
        return this._projectService.updateProject(project).pipe(map(res => {
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
        }));
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
        return this._projectService.connectRepoManager(key, repoName).pipe(map( res => {
            let cache = this._projectCache.getValue();
            if (cache.get(key)) {
                let pToUpdate = cache.get(key);
                pToUpdate.last_modified = res.last_modified;
                this._projectCache.next(cache.set(key, pToUpdate));
            }
            return res;
        }));
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
        return this._projectService.callback(key, repoName, token, verifier).pipe(map( res => {
            let cache = this._projectCache.getValue();
            let projectToUpdate = cache.get(key);
            if (projectToUpdate) {
                projectToUpdate.last_modified = res.last_modified;
                projectToUpdate.vcs_servers = res.vcs_servers;
                this._projectCache.next(cache.set(key, projectToUpdate));
            }
            return res;
        }));
    }

    /**
     * Disconnect a repo manager from the given project.
     * @param key Project unique key
     * @param repoName Repo manager to disconnect
     * @returns {Observable<Project>}
     */
    disconnectRepoManager(key: string, repoName: string): Observable<Project> {
        return this._projectService.disconnectRepoManager(key, repoName).pipe(map( res => {
            let cache = this._projectCache.getValue();
            let pToUpdate = cache.get(key);
            if (pToUpdate) {
                pToUpdate.last_modified = res.last_modified;
                if (pToUpdate.vcs_servers) {
                    let indexRepo: number = pToUpdate.vcs_servers.findIndex(r => r.name === repoName);
                    if (indexRepo >= 0) {
                        pToUpdate.vcs_servers.splice(indexRepo, 1);
                        this._projectCache.next(cache.set(key, pToUpdate));
                    }
                }
            }
            return res;
        }));
    }

    /**
     * Delete the given project
     * @param key Project unique key
     * @returns {Observable<boolean>}
     */
    deleteProject(key: string): Observable<boolean> {
        return this._projectService.deleteProject(key).pipe(map(res => {
            let projects = this._projectNav.getValue();
            let index = projects.findIndex(prj => prj.key === key);
            this._projectNav.next(projects.delete(index));

            this.removeFromStore(key);
            return res;
        }));
    }

    removeFromStore(key: string) {
        let cache = this._projectCache.getValue();
        this._projectCache.next(cache.delete(key));
    }

    /**
     * Add a variable for the given project
     * @param key Project unique key
     * @param variable Variable to add
     * @returns {Observable<Project>}
     */
    addProjectVariable(key: string, variable: Variable): Observable<Project> {
        return this._projectService.addVariable(key, variable).pipe(map(res => {
            return this.refreshProjectVariableCache(key, res);
        }));
    }

    /**
     * Update a variable for the given project
     * @param key Project unique key
     * @param variable Variable to update
     * @returns {Observable<Project>}
     */
    updateProjectVariable(key: string, variable: Variable): Observable<Variable> {
        return this._projectService.updateVariable(key, variable).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let varIndex = projectUpdate.variables.findIndex(v => v.id === res.id);
                if (varIndex > -1) {
                    projectUpdate.variables[varIndex] = res;
                    this._projectCache.next(cache.set(key, projectUpdate));
                }
            }
            return res;
        }));
    }

    /**
     * Delete a variable for the given project
     * @param key Project unique key
     * @param variable Variable to delete
     * @returns {Observable<Project>}
     */
    deleteProjectVariable(key: string, variable: Variable): Observable<boolean> {
        return this._projectService.removeVariable(key, variable.name).pipe(map(() => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let varIndex = projectUpdate.variables.findIndex(v => v.id === variable.id);
                if (varIndex > -1) {
                    projectUpdate.variables.splice(varIndex, 1);
                    this._projectCache.next(cache.set(key, projectUpdate));
                }
            }
            return true;
        }));
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
    addProjectPermission(key: string, gp: GroupPermission): Observable<Array<GroupPermission>> {
        return this._projectService.addPermission(key, gp).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                projectUpdate.groups = res;
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        }));
    }

    /**
     * Update a group permission
     * @param key Project unique key
     * @param gp Permission to update
     * @returns {Observable<Project>}
     */
    updateProjectPermission(key: string, gp: GroupPermission): Observable<GroupPermission> {
        gp.permission = Number(gp.permission);
        return this._projectService.updatePermission(key, gp).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let permissionIndex = projectUpdate.groups.findIndex( p => p.group.id === res.group.id);
                if (permissionIndex > -1) {
                    delete gp.hasChanged;
                    delete gp.updating;
                    projectUpdate.groups[permissionIndex] = res;
                }
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        }));
    }

    /**
     * Remove a group permission
     * @param key Project unique key
     * @param gp Permission to delete
     * @returns {Observable<Project>}
     */
    removeProjectPermission(key: string, gp: GroupPermission): Observable<boolean> {
        return this._projectService.removePermission(key, gp).pipe(map(() => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                projectUpdate.groups = projectUpdate.groups.filter( p => p.group.id !== gp.group.id);
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return true;
        }));
    }

    /**
     * Add an environment
     * @param key Project unique key
     * @param environment Environment to add
     */
    addProjectEnvironment(key: string, environment: Environment) {
        return this._projectService.addEnvironment(key, environment).pipe(map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        }));
    }

    /**
     * Rename an environment
     * @param key Project unique key
     * @param environment Environment to rename
     */
    renameProjectEnvironment(key: string, oldName: string, environment: Environment) {
        return this._projectService.renameEnvironment(key, oldName, environment).pipe(map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        }));
    }

    /**
     * Clone an environment
     * @param key Project unique key
     * @param environment Environment to clone
     * @param cloneName for the new environment cloned
     */
    cloneProjectEnvironment(key: string, environment: Environment, cloneName: string) {
        return this._projectService.cloneEnvironment(key, environment, cloneName).pipe(map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        }));
    }

    /**
     * Remove an environment
     * @param key Project unique key
     * @param environment Environment to rename
     */
    deleteProjectEnvironment(key: string, environment: Environment) {
        return this._projectService.removeEnvironment(key, environment).pipe(map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        }));
    }

    /**
     * Add a variable in an environment
     * @param key Project unique key
     * @param envName Environment Name
     * @param v variable to add
     * @returns {Observable<Project>}
     */
    addEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._projectService.addEnvironmentVariable(key, envName, v).pipe(map(res => {
           return this.refreshProjectEnvironmentCache(key, res);
        }));
    }

    /**
     * Update a variable in an environment
     * @param key Project unique key
     * @param envName Environment Name
     * @param v variable to update
     * @returns {Observable<Project>}
     */
    updateEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._projectService.updateEnvironmentVariable(key, envName, v).pipe(map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        }));
    }

    /**
     * Remove a variable from an environment
     * @param key Project unique key
     * @param envName Environment Name
     * @param v variable to remove
     * @returns {Observable<Project>}
     */
    removeEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._projectService.removeEnvironmentVariable(key, envName, v).pipe(map(res => {
            return this.refreshProjectEnvironmentCache(key, res);
        }));
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

    /**
     * Add environment permission
     * @param key project unique key
     * @param envName Environment name
     * @param gps Group permission to add
     * @returns {Observable<Environment>}
     */
    addEnvironmentPermission(key: string, envName: string, gps: Array<GroupPermission>): Observable<Project> {
        return this._projectService.addEnvironmentPermission(key, envName, gps).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let index = projectUpdate.environments.findIndex(env => env.id === res.id);
                projectUpdate.environments[index] = res;
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return projectUpdate;
        }));
    }

    /**
     * Update environment permission
     * @param key Project unique key
     * @param envName Environment Name
     * @param gp Group permission to update
     * @returns {Observable<Environmenet>}
     */
    updateEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<Environment> {
        return this._projectService.updateEnvironmentPermission(key, envName, gp).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let index = projectUpdate.environments.findIndex(env => env.id === res.id);
                projectUpdate.environments[index] = res;
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        }));
    }

    /**
     * Remove a permission from an environment
     * @param key Project unique key
     * @param envName Environment name
     * @param gp Permission to remove
     * @returns {Observable<boolean>}
     */
    removeEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<boolean> {
        return this._projectService.removeEnvironmentPermission(key, envName, gp).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let e = projectUpdate.environments.find(env => env.name === envName);
                e.groups = e.groups.filter(groupPermission => groupPermission.group.id !== gp.group.id);
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        }));
    }

    /**
     * Ad a key on the project
     * @param projKey Project unique key
     * @param key SSH/PGP key to add
     * @returns {Observable<Key>}
     */
    addKey(projKey: string, key: Key): Observable<Key> {
        return this._projectService.addKey(projKey, key).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(projKey);
            if (projectUpdate) {
                if (!projectUpdate.keys) {
                    projectUpdate.keys = new Array<Key>();
                }
                projectUpdate.keys.push(res);
                this._projectCache.next(cache.set(projKey, projectUpdate));
            }
            return res;
        }));
    }

    /**
     * Remove a key from project (api + cache)
     * @param key project unique key
     * @param name key name to delete
     * @returns {Observable<boolean>}
     */
    removeKey(key: string, name: string): Observable<boolean> {
        return this._projectService.removeKey(key, name).pipe(map(() => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate && projectUpdate.keys) {
                let i = projectUpdate.keys.findIndex(kkey => kkey.name === name);
                if (i > -1) {
                    projectUpdate.keys.splice(i, 1);
                }
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return true;
        }));
    }

    externalModification(key: string) {
        let cache = this._projectCache.getValue();
        let projectUpdate = cache.get(key);
        if (projectUpdate) {
            projectUpdate.externalChange = true;
            this._projectCache.next(cache.set(key, projectUpdate));
        }
    }

    /**
     * Use by router to preload project
     * @param key
     * @returns {Observable<Project>}
     */
    getProjectIntegrationsResolver(key: string): Observable<Project> {
        let store = this._projectCache.getValue();
        let missingIntegrations = store.size === 0
            || !store.get(key)
            || !store.get(key).integrations
            || !store.get(key).integrations.length;

        if (missingIntegrations) {
            return this.resyncIntegrations(key);
        } else {
            return observableOf(store.get(key));
        }
    }

    /**
     * Get project integrations
     * @param key
     * @returns {Observable<R>}
     */
    resyncIntegrations(key: string): Observable<Project> {
        return this._projectService.getIntegrations(key).pipe(
            map((res) => {
                let store = this._projectCache.getValue();
                let proj = store.get(key);
                proj.integrations = res;
                this._projectCache.next(store.set(key, proj));
                return proj;
            }));
    }

    /**
     * Add a integration to a project
     * @param key Project unique key
     * @param integration Integration to add
     */
    addIntegration(key: string, integration: ProjectIntegration): Observable<ProjectIntegration> {
        return this._projectService.addIntegration(key, integration).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                if (!projectUpdate.integrations) {
                    projectUpdate.integrations = new Array<ProjectIntegration>();
                }
                projectUpdate.integrations.push(res);
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        }));
    }

    deleteProjectIntegration(key: string, integrationName: string) {
        return this._projectService.removeIntegration(key, integrationName).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                if (!projectUpdate.integrations) {
                    return res;
                }
                projectUpdate.integrations = projectUpdate.integrations.filter(p => p.name !== integrationName);
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        }));
    }

    updateProjectIntegration(key: string, integration: ProjectIntegration): Observable<ProjectIntegration> {
        return this._projectService.updateIntegration(key, integration).pipe(map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                if (!projectUpdate.integrations) {
                    return res;
                }
                let index = projectUpdate.integrations.findIndex(p => p.name === integration.name);
                if (index !== -1) {
                    projectUpdate.integrations[index] = res;
                }
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        }));
    }

    updateLabels(key: string, labels: Label[]): Observable<Project> {
        return this._projectService.updateLabels(key, labels)
          .pipe(
            map((proj) => {
              let cache = this._projectCache.getValue();
              let projectUpdate = cache.get(key);
              if (projectUpdate) {
                  projectUpdate.labels = proj.labels;
                  projectUpdate.workflow_names = proj.workflow_names;
                  this._projectCache.next(cache.set(key, projectUpdate));
              }
              return projectUpdate;
            })
          );
    }

}
