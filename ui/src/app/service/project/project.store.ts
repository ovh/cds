import {Injectable} from '@angular/core';
import {List, Map} from 'immutable';
import {Observable} from 'rxjs/Observable';
import {BehaviorSubject} from 'rxjs/BehaviorSubject'
import {Project} from '../../model/project.model';
import {ProjectService} from './project.service';
import {EnvironmentService} from '../environment/environment.service';
import {PipelineService} from '../pipeline/pipeline.service';
import {VariableService} from '../variable/variable.service';
import {Variable} from '../../model/variable.model';
import {GroupPermission} from '../../model/group.model';
import {Environment} from '../../model/environment.model';

@Injectable()
export class ProjectStore {

    // List of all project. Use by Navbar
    private _projectNav: BehaviorSubject<List<Project>> = new BehaviorSubject(List([]));

    // List of all project + dependencies:  List of variables, List of Env, List of App, List of Pipeline.
    private _projectCache: BehaviorSubject<Map<string, Project>> = new BehaviorSubject(Map<string, Project>());

    constructor(
        private _projectService: ProjectService,
        private _environmentService: EnvironmentService,
        private _pipelineService: PipelineService,
        private _variableService: VariableService
      ) {

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
    getProjectResolver(key: string, opts: LoadOpts[]): Observable<Project> {
        let store = this._projectCache.getValue();
        if (store.size === 0 || !store.get(key)) {
            opts = opts.concat([
                new LoadOpts('withGroups', 'groups'),
                new LoadOpts('withPermission', 'permission')
            ]);
            return this.resync(key, opts);
        }

        if (Array.isArray(opts) && store.get(key)) {
            let funcs = opts.filter((opt) => store.get(key)[opt.fieldName] == null);

            if (!funcs.length) {
                return Observable.of(store.get(key));
            }

            funcs = funcs.concat([
                new LoadOpts('withGroups', 'groups'),
                new LoadOpts('withPermission', 'permission')
            ]);

            // TODO: iterate on funcs array and execute handler linked to fetch missing data
            return this.resync(key, funcs);
        }
        return Observable.of(store.get(key));
    }

    /**
     * Get project from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resync(key: string, opts?: LoadOpts[]): Observable<Project> {
        return this._projectService.getProject(key, opts).map( res => {
            let store = this._projectCache.getValue();
            let proj = store.get(key);
            if (proj) {
                proj = Object.assign({}, proj, res);
            } else {
                proj = res;
            }

            this._projectCache.next(store.set(key, proj));
            return proj;
        });
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
            return Observable.of(store.get(key));
        }
    }

    /**
     * Get project from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resyncEnvironments(key: string): Observable<Project> {
        return this._environmentService.get(key)
          .map((res) => {
              let store = this._projectCache.getValue();
              let proj = store.get(key);
              proj.environments = res;
              this._projectCache.next(store.set(key, proj));
              return proj;
          });
    }

    /**
     * Use by router to preload project
     * @param key
     * @returns {Observable<Project>}
     */
    getProjectPipelinesResolver(key: string): Observable<Project> {
        let store = this._projectCache.getValue();
        let missingEnv = store.size === 0 || !store.get(key) || !store.get(key).pipelines || !store.get(key).pipelines.length;

        if (missingEnv) {
            return this.resyncPipelines(key);
        } else {
            return Observable.of(store.get(key));
        }
    }

    /**
     * Get project from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resyncPipelines(key: string): Observable<Project> {
        return this._pipelineService.getPipelines(key)
          .map((res) => {
              let store = this._projectCache.getValue();
              let proj = store.get(key);
              proj.pipelines = res;
              this._projectCache.next(store.set(key, proj));
              return proj;
          });
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
            return Observable.of(store.get(key));
        }
    }

    /**
     * Get project from API and store result
     * @param key
     * @returns {Observable<R>}
     */
    resyncVariables(key: string): Observable<Project> {
        return this._variableService.get(key)
          .map((res) => {
              let store = this._projectCache.getValue();
              let proj = store.get(key);
              proj.variables = res;
              this._projectCache.next(store.set(key, proj));
              return proj;
          });
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
     * Update applications and pipelines on a project
     * @param key project key
     * @param project project to update
     */
    updateApplicationsAndPipelines(key: string, project: Project): void {
        let cache = this._projectCache.getValue();
        let projectToUpdate = cache.get(key);
        if (projectToUpdate) {
            projectToUpdate.applications = project.applications;
            projectToUpdate.pipelines = project.pipelines;
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
                pToUpdate.last_modified = res.last_modified;
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
                projectToUpdate.vcs_servers = res.vcs_servers;
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
                if (pToUpdate.vcs_servers) {
                    let indexRepo: number = pToUpdate.vcs_servers.findIndex(r => r.name === repoName);
                    if (indexRepo >= 0) {
                        pToUpdate.vcs_servers.splice(indexRepo, 1);
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

            this.removeFromStore(key);
            return res;
        });
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
    updateProjectVariable(key: string, variable: Variable): Observable<Variable> {
        return this._projectService.updateVariable(key, variable).map(res => {
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
        });
    }

    /**
     * Delete a variable for the given project
     * @param key Project unique key
     * @param variable Variable to delete
     * @returns {Observable<Project>}
     */
    deleteProjectVariable(key: string, variable: Variable): Observable<boolean> {
        return this._projectService.removeVariable(key, variable.name).map(() => {
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
    addProjectPermission(key: string, gp: GroupPermission): Observable<Array<GroupPermission>> {
        return this._projectService.addPermission(key, gp).map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                projectUpdate.groups = res;
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        });
    }

    /**
     * Update a group permission
     * @param key Project unique key
     * @param gp Permission to update
     * @returns {Observable<Project>}
     */
    updateProjectPermission(key: string, gp: GroupPermission): Observable<GroupPermission> {
        gp.permission = Number(gp.permission);
        return this._projectService.updatePermission(key, gp).map(res => {
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
        });
    }

    /**
     * Remove a group permission
     * @param key Project unique key
     * @param gp Permission to delete
     * @returns {Observable<Project>}
     */
    removeProjectPermission(key: string, gp: GroupPermission): Observable<boolean> {
        return this._projectService.removePermission(key, gp).map(() => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                projectUpdate.groups = projectUpdate.groups.filter( p => p.group.id !== gp.group.id);
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return true;
        });
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
     * Clone an environment
     * @param key Project unique key
     * @param environment Environment to clone
     * @param cloneName for the new environment cloned
     */
    cloneProjectEnvironment(key: string, environment: Environment, cloneName: string) {
        return this._projectService.cloneEnvironment(key, environment, cloneName).map(res => {
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

    /**
     * Add environment permission
     * @param key project unique key
     * @param envName Environment name
     * @param gps Group permission to add
     * @returns {Observable<Environment>}
     */
    addEnvironmentPermission(key: string, envName: string, gps: Array<GroupPermission>): Observable<Project> {
        return this._projectService.addEnvironmentPermission(key, envName, gps).map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let index = projectUpdate.environments.findIndex(env => env.id === res.id);
                projectUpdate.environments[index] = res;
                this._projectCache.next(cache.set(key, projectUpdate));
                return projectUpdate;
            }
        });
    }

    /**
     * Update environment permission
     * @param key Project unique key
     * @param envName Environment Name
     * @param gp Group permission to update
     * @returns {Observable<Environmenet>}
     */
    updateEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<Environment> {
        return this._projectService.updateEnvironmentPermission(key, envName, gp).map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let index = projectUpdate.environments.findIndex(env => env.id === res.id);
                projectUpdate.environments[index] = res;
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        });
    }

    /**
     * Remove a permission from an environment
     * @param key Project unique key
     * @param envName Environment name
     * @param gp Permission to remove
     * @returns {Observable<boolean>}
     */
    removeEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<boolean> {
        return this._projectService.removeEnvironmentPermission(key, envName, gp).map(res => {
            let cache = this._projectCache.getValue();
            let projectUpdate = cache.get(key);
            if (projectUpdate) {
                let e = projectUpdate.environments.find(env => env.name === envName);
                e.groups = e.groups.filter(groupPermission => groupPermission.group.id !== gp.group.id);
                this._projectCache.next(cache.set(key, projectUpdate));
            }
            return res;
        });
    }


    externalModification(key: string) {
        let cache = this._projectCache.getValue();
        let projectUpdate = cache.get(key);
        if (projectUpdate) {
            projectUpdate.externalChange = true;
            this._projectCache.next(cache.set(key, projectUpdate));
        }
    }


}

export class LoadOpts {
  constructor(
    public queryParam: string,
    public fieldName: string
  ) { }
}
