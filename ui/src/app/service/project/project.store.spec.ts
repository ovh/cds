/* tslint:disable:no-unused-variable */

import {async, TestBed} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {AppModule} from '../../app.module';
import {RouterModule} from '@angular/router';
import {ProjectStore} from './project.store';
import {LoadOpts, Project} from '../../model/project.model';
import {Variable} from '../../model/variable.model';
import {Group, GroupPermission} from '../../model/group.model';
import {Environment} from '../../model/environment.model';
import {Application} from '../../model/application.model';
import {RepositoriesManager} from '../../model/repositories.model';
import {first} from 'rxjs/operators';
import {Observable} from 'rxjs/Observable';
import {ProjectService} from './project.service';

describe('CDS: project Store', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: APP_BASE_HREF, useValue: '/'},
                {provide: ProjectService, useClass: MockProjectService}
            ],
            imports: [
                AppModule,
                RouterModule
            ]
        });

    });

    it('should Get Projects', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        // Get projects from HTTP Call
        projectStore.getProjectsList().pipe(first()).subscribe(() => {
        });

        projectStore.getProjectsList().pipe(first()).subscribe(pdata => {
            expect(pdata.size).toBe(4, 'Wrong number of project. Must be 4.');
        });
    }));

    it('should create a Project', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        let project = createProject('key1', 'myProject');
        let project2 = createProject('key2', 'myProject2');

        // Create project
        projectStore.createProject(createProject('key1', 'myProject')).subscribe(() => {
        });

        projectStore.createProject(createProject('key2', 'myProject2')).subscribe(() => {
        });

        // check cache for nav
        let checkedNav = false;
        projectStore.getProjectsList().pipe(first()).subscribe(projects => {
            expect(projects.size).toBe(2, 'Wrong number of project. Must be 2.');
            checkedNav = true;
        });
        expect(checkedNav).toBeTruthy('Need at least 2 projects in nav');

        // check get project: first time ( get from http call)
        projectStore.getProjects('key1').subscribe(projs => {
        });
        let checkedSingleProject = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            expect(projs.get('key1').key).toBe('key1', 'Wrong project key. Must be key1');
            checkedSingleProject = true;
        });
        expect(checkedSingleProject).toBeTruthy('Need to get project key1');


        // Delete project
        projectStore.deleteProject('key1').subscribe(() => {
        });

        let checkedDeleteProject = false;
        projectStore.getProjects('key1').subscribe(() => {
            checkedDeleteProject = true;
        });
    }));

    it('should update Project', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        // Create project
        let p = createProject('key1', 'myProject');
        projectStore.createProject(p).subscribe(() => {
        });

        // check get project: first time ( get from http call)
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {

        });

        // Update
        p.name = 'myProjectUpdate1';
        projectStore.updateProject(p).subscribe(() => {
        });

        // check cache for nav
        let checkedNav = false;
        projectStore.getProjectsList().pipe(first()).subscribe(projects => {
            expect(projects.size).toBe(1, 'Wrong number of project. Must be 1.');
            expect(projects.get(0).name).toBe('myProjectUpdate1', 'Wrong name of project. Must be myProjectUpdate1');
            checkedNav = true;
        });
        expect(checkedNav).toBeTruthy('Need at least 2 projects in nav');

        let checkedSingleProject = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            expect(projs.get('key1').name).toBe('myProjectUpdate1', 'Wrong project name. Must be myProjectUpdate1');
            checkedSingleProject = true;
        });
        expect(checkedSingleProject).toBeTruthy('Need to get project key1');

        // Update 2
        p.name = 'myProjectUpdate2';
        projectStore.updateProject(p).subscribe(() => {
        });

        // check get project: second time (get from cache
        let checkedSingleProjectTwice = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(pp => {
            expect(pp.get('key1').name).toBe('myProjectUpdate2', 'Wrong project name. Must be myProjectUpdate2');
            checkedSingleProjectTwice = true;
        });
        expect(checkedSingleProjectTwice).toBeTruthy('Need to get project key1 twice');
    }));

    it('should add a repomananger + validate repo manager + delete repomanager', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        let projectValidation = createProject('key1', 'myProject');
        projectValidation.last_modified = '456';
        projectValidation.vcs_servers = new Array<RepositoriesManager>();
        let rp = new RepositoriesManager();
        rp.name = 'repoName';
        projectValidation.vcs_servers.push(rp);

        let projectRepoDel = createProject('key1', 'myProject');
        projectRepoDel.last_modified = '789';


        // Get project in cache
        projectStore.getProjects('key1').pipe(first()).subscribe(() => {
        });


        // Add repo
        projectStore.connectRepoManager('key1', 'repoName').subscribe(() => {
        });

        let addRepoCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addRepoCheck = true;
        });
        expect(addRepoCheck).toBeTruthy('Must check project update');

        // validate repo
        projectStore.verificationCallBackRepoManager('key1', 'repoName', 'token', 'code').subscribe(() => {
        });

        let validationRepoCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            validationRepoCheck = true;
            expect(projs.get('key1').vcs_servers.length).toBe(1);
        });
        expect(validationRepoCheck).toBeTruthy('Must check project update');

        projectStore.disconnectRepoManager('key1', 'repoName').subscribe((p) => {
        });

        let deleteRepoCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteRepoCheck = true;
            expect(projs.get('key1').vcs_servers.length).toBe(0);
        });
        expect(deleteRepoCheck).toBeTruthy('Must check project update');
    }));

    it('should add/update/delete a variable', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        let projectAdd = createProject('key1', 'myProject');
        projectAdd.last_modified = '0';

        let projectAddVar = createProject('key1', 'myProject');
        projectAddVar.last_modified = '123';
        let vAdd = new Variable();
        vAdd.name = 'myvar';
        projectAddVar.variables = new Array<Variable>();
        projectAddVar.variables.push(vAdd);

        let projectUpVar = createProject('key1', 'myProject');
        let vUp = new Variable();
        vUp.name = 'myvarUpdate';
        projectUpVar.variables = new Array<Variable>();
        projectUpVar.variables.push(vUp);


        let projectDelVar = createProject('key1', 'myProject');
        projectDelVar.variables = new Array<Variable>();

        // Get project in cache
        projectStore.getProjects('key1').pipe(first()).subscribe(() => {
        });

        // Add variable
        let v: Variable = new Variable();
        v.name = 'var1';
        v.type = 'string';
        v.value = 'value1';
        v.description = 'myDescription';
        projectStore.addProjectVariable('key1', v).subscribe(() => {
        });

        let addVariableCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addVariableCheck = true;
            expect(projs.get('key1').variables.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').variables[0].name).toBe('var1');
        });
        expect(addVariableCheck).toBeTruthy('Must check project update');


        // update variable
        v.name = 'myvarUpdate';
        projectStore.updateProjectVariable('key1', v).subscribe(() => {
        });

        let updateVariableCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            updateVariableCheck = true;
            expect(projs.get('key1').variables.length).toBe(1);
            expect(projs.get('key1').variables[0].name).toBe('myvarUpdate');
        });
        expect(updateVariableCheck).toBeTruthy('Must check project update');

        // Delete variable
        projectStore.deleteProjectVariable('key1', v).subscribe(() => {
        });

        let deleteVariableCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteVariableCheck = true;
            expect(projs.get('key1').variables.length).toBe(0);
        });
        expect(deleteVariableCheck).toBeTruthy('Must check project update');
    }));

    it('should add/update/delete a permission', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        let grp = new Group();
        grp.id = 1;
        grp.name = 'grp';

        let projectAdd = createProject('key1', 'myproject');

        let projectAddGrp = createProject('key1', 'myproject');
        projectAddGrp.groups = new Array<GroupPermission>();
        let gpAdd = new GroupPermission();
        gpAdd.group = grp;
        gpAdd.permission = 7;
        projectAddGrp.groups.push(gpAdd);

        let projectUpGrp = createProject('key1', 'myproject');
        projectUpGrp.groups = new Array<GroupPermission>();
        let gpUp = new GroupPermission();
        gpUp.group = grp;
        gpUp.permission = 4;
        projectUpGrp.groups.push(gpUp);

        let projectDelGrp = createProject('key1', 'myproject');
        projectDelGrp.groups = new Array<GroupPermission>();

        // Get project in cache
        projectStore.getProjects('key1').subscribe(() => {
        });

        // Add groupPermission
        let gp: GroupPermission = new GroupPermission();
        gp.group = new Group();
        gp.group.id = 1;
        gp.group.name = 'mygroup';
        gp.permission = 7;
        projectStore.addProjectPermission('key1', gp).subscribe(() => {
        });

        let addPermissionCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addPermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(1, 'Project must have 1 group');
            expect(projs.get('key1').groups[0].permission).toBe(7);
        });
        expect(addPermissionCheck).toBeTruthy('Must check project update');


        // update permission
        gp.permission = 4;
        projectStore.updateProjectPermission('key1', gp).subscribe(() => {
        });

        let updatePermissionCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            updatePermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(1);
            expect(projs.get('key1').groups[0].permission).toBe(4);
        });
        expect(updatePermissionCheck).toBeTruthy('Must check project update');

        // Delete group permission
        gp.group.id = 1;
        projectStore.removeProjectPermission('key1', gp).subscribe(() => {
        });

        let deletePermissionCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deletePermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(0);
        });
        expect(deletePermissionCheck).toBeTruthy('Must check project update');
    }));

    it('should add/update/delete an environment', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        let projectAdd = createProject('key1', 'myProject');

        let projectAddEnv = createProject('key1', 'myProject');
        projectAddEnv.last_modified = '123';
        projectAddEnv.environments = new Array<Environment>();
        let envAdd = new Environment();
        envAdd.name = 'Production';
        projectAddEnv.environments.push(envAdd);

        let projectUpEnv = createProject('key1', 'myProject');
        projectUpEnv.last_modified = '456';
        projectUpEnv.environments = new Array<Environment>();
        let envUp = new Environment();
        envUp.name = 'PreProduction';
        projectUpEnv.environments.push(envUp);

        let projectDelEnv = createProject('key1', 'myProject');
        projectDelEnv.last_modified = '789';
        projectDelEnv.environments = new Array<Environment>();

        // Get project in cache
        projectStore.getProjects('key1').subscribe(() => {
        });

        // Add evn
        let env: Environment = new Environment();
        env.name = 'Production';
        projectStore.addProjectEnvironment('key1', env).subscribe(() => {
        });

        let addEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').environments[0].name).toBe('Production');
        });
        expect(addEnvCheck).toBeTruthy('Must check project update');


        // update env
        env.name = 'PreProduction';
        projectStore.renameProjectEnvironment('key1', env.name, env).subscribe(() => {
        });

        let renameEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            renameEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].name).toBe('PreProduction');
        });
        expect(renameEnvCheck).toBeTruthy('Must check project update');

        // Delete env
        env.name = 'PreProduction';
        projectStore.deleteProjectEnvironment('key1', env).subscribe(() => {
        });

        let deleteEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(0);
        });
        expect(deleteEnvCheck).toBeTruthy('Must check project delete env');
    }));

    it('should add/update/delete an environment variable', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        let projectAdd = createProject('key1', 'myProject');
        projectAdd.environments = new Array<Environment>();
        let envAdd = new Environment();
        envAdd.name = 'prod';
        projectAdd.environments.push(envAdd);

        let projectAddEnvVar = createProject('key1', 'myProject');
        projectAddEnvVar.last_modified = '123';
        projectAddEnvVar.environments = new Array<Environment>();
        let envAddVar = new Environment();
        envAddVar.name = 'prod';
        envAddVar.variables = new Array<Variable>();
        let vAdd = new Variable();
        vAdd.name = 'foo';
        envAddVar.variables.push(vAdd);
        projectAddEnvVar.environments.push(envAddVar);

        let projectUpEnvVar = createProject('key1', 'myProject');
        projectUpEnvVar.last_modified = '456';
        projectUpEnvVar.environments = new Array<Environment>();
        let envUpVar = new Environment();
        envUpVar.name = 'prod';
        envUpVar.variables = new Array<Variable>();
        let vUp = new Variable();
        vUp.name = 'fooUpdated';
        envUpVar.variables.push(vUp);
        projectUpEnvVar.environments.push(envUpVar);

        let projectDelEnvVar = createProject('key1', 'myProject');
        projectDelEnvVar.last_modified = '789';
        projectDelEnvVar.environments = new Array<Environment>();
        let envDeL = new Environment();
        envDeL.name = 'prod';
        envDeL.variables = new Array<Variable>();
        projectDelEnvVar.environments.push(envDeL);

        // Get project in cache
        projectStore.getProjects('key1').subscribe(() => {
        });

        // Add env variable
        let v: Variable = new Variable();
        v.name = 'foo';
        projectStore.addEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        });

        let addEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').environments[0].variables.length).toBe(1);
            expect(projs.get('key1').environments[0].variables[0].name).toBe('foo');
        });
        expect(addEnvCheck).toBeTruthy('Must check project update');


        // update var
        v.name = 'fooUpdated';
        projectStore.updateEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        });

        let renameVarEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            renameVarEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].variables.length).toBe(1);
            expect(projs.get('key1').environments[0].variables[0].name).toBe('fooUpdated');
        });
        expect(renameVarEnvCheck).toBeTruthy('Must check project update');

        // Delete var
        projectStore.removeEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        });

        let deleteVarCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteVarCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].variables.length).toBe(0);
        });
        expect(deleteVarCheck).toBeTruthy('Must check project delete var');
    }));

    it('should add/update/delete an environment permission', async(() => {
        const projectStore = TestBed.get(ProjectStore);

        let envAdd = new Environment();
        envAdd.name = 'prod';

        // Get project in cache
        projectStore.getProjects('key1').subscribe(() => {
        });

        projectStore.addProjectEnvironment('key1', envAdd).subscribe(() => {});

        // Add env permission
        let gpA: Array<GroupPermission> = new Array<GroupPermission>();
        let gp = new GroupPermission();
        gp.permission = 7;
        gp.group = new Group();
        gp.group.id = 1;
        gp.group.name = 'grp1';
        gpA.push(gp);
        projectStore.addEnvironmentPermission('key1', envAdd.name, gpA).subscribe(() => {});

        let addEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 env');
            expect(projs.get('key1').environments[0].groups.length).toBe(1);
            expect(projs.get('key1').environments[0].groups[0].permission).toBe(7);
            expect(projs.get('key1').environments[0].groups[0].group.name).toBe('grp1');
        });
        expect(addEnvCheck).toBeTruthy('Must check env update');

        // update gp
        gp.permission = 4;
        projectStore.updateEnvironmentPermission('key1', envAdd.name, gp).subscribe(() => {
        });

        let renameVarEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            renameVarEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].groups.length).toBe(1);
            expect(projs.get('key1').environments[0].groups[0].permission).toBe(4);
        });
        expect(renameVarEnvCheck).toBeTruthy('Must check env update');
    }));


    function createProject(key: string, name: string): Project {
        let project: Project = new Project();
        project.name = name;
        project.key = key;
        return project;
    }

    function createApplication(name: string): Application {
        let app = new Application();
        app.name = name;
        return app;
    }

    class MockProjectService {

        getProjects(withApplication: boolean): Observable<Project[]> {
            let projects = new Array<Project>();
            projects.push(createProject('key1', 'myProject'));
            projects.push(createProject('key2', 'myProject2'));
            projects.push(createProject('key3', 'myProject3'));
            projects.push(createProject('key4', 'myProject4'));
            return Observable.of(projects);
        }

        addProject(project: Project): Observable<Project> {
            project.name = 'myProject';
            project.applications = new Array<Application>();
            project.applications.push(createApplication('app1'));
            project.applications.push(createApplication('app2'));
            project.applications.push(createApplication('app3'));
            return Observable.of(project);
        }

        updateProject(project: Project): Observable<Project> {
            return Observable.of(project);
        }

        getProject(key: string, opts: LoadOpts[]): Observable<Project> {
            let p = new Project();
            p.key = key;
            return Observable.of(p);
        }

        addVariable(key: string, v: Variable): Observable<Project> {
            let p = new Project();
            p.key = key;
            p.variables = new Array<Variable>();
            p.variables.push(v);
            return Observable.of(p);
        }

        updateVariable(key: string, v: Variable): Observable<Variable> {
            return Observable.of(v);
        }

        removeVariable(key: string, varName: string): Observable<boolean> {
            return Observable.of(true);
        }

        addPermission(key: string, gp: GroupPermission): Observable<Array<GroupPermission>> {
            let gps = new Array<GroupPermission>();
            gps.push(gp);
            return Observable.of(gps);
        }

        updatePermission(key: string, gp: GroupPermission): Observable<GroupPermission> {
            return Observable.of(gp);
        }

        removePermission(key: string, gp: GroupPermission): Observable<boolean> {
            return Observable.of(true);
        }

        connectRepoManager(key: string, repoName: string): Observable<any> {
            return Observable.of({
                'project_key': 'key1',
                'last_modified': '123',
                'repositories_manager': 'repoName',
                'url': 'ff/.c',
                'request_token': 'mytoken'
            });
        }

        disconnectRepoManager(key: string, repoName: string): Observable<Project> {
            let p = new Project();
            p.vcs_servers = new Array<RepositoriesManager>();
            return Observable.of(p);
        }

        callback(key: string, repoName: string, token: string, verifier: string): Observable<Project> {
            let p = new Project();
            p.vcs_servers = new Array<RepositoriesManager>();
            let r = new RepositoriesManager();
            r.name = repoName;
            p.vcs_servers.push(r);
            return Observable.of(p);
        }

        addEnvironment(key: string, environment: Environment): Observable<Project> {
            let p = new Project();
            p.environments = new Array<Environment>();
            p.environments.push(environment);
            return Observable.of(p);
        }

        renameEnvironment(key: string, oldName: string, environment: Environment): Observable<Project> {
            let p = new Project();
            p.environments = new Array<Environment>();
            p.environments.push(environment);
            return Observable.of(p);
        }

        removeEnvironment(key: string, environment: Environment): Observable<Project> {
            let p = new Project();
            p.environments = new Array<Environment>();
            return Observable.of(p);
        }

        addEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
            let p = new Project();
            p.environments = new Array<Environment>();

            let e = new Environment();
            p.environments.push(e);
            e.variables = new Array<Variable>();
            e.variables.push(v);
            return Observable.of(p);
        }

        updateEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
            let p = new Project();
            p.environments = new Array<Environment>();

            let e = new Environment();
            p.environments.push(e);
            e.variables = new Array<Variable>();
            e.variables.push(v);
            return Observable.of(p);
        }

        removeEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
            let p = new Project();
            p.environments = new Array<Environment>();

            let e = new Environment();
            p.environments.push(e);
            e.variables = new Array<Variable>();
            return Observable.of(p);
        }

        addEnvironmentPermission(key: string, envName: string, gps: Array<GroupPermission>): Observable<Environment> {
            let e = new Environment();
            e.groups = gps
            return Observable.of(e);
        }

        updateEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<Environment> {
            let e = new Environment();
            e.groups = new Array<GroupPermission>();
            e.groups.push(gp);
            return Observable.of(e);
        }

        removeEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<boolean> {
            return Observable.of(true);
        }

        deleteProject(key: string): Observable<boolean> {
            return Observable.of(true);
        }
    }
});
