/* tslint:disable:no-unused-variable */

import {async, TestBed} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {AppModule} from '../../app.module';
import {RouterModule} from '@angular/router';
import {ProjectStore} from './project.store';
import {Project} from '../../model/project.model';
import {Variable} from '../../model/variable.model';
import {Group, GroupPermission} from '../../model/group.model';
import {Environment} from '../../model/environment.model';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import {Application} from '../../model/application.model';
import {RepositoriesManager} from '../../model/repositories.model';
import {first} from 'rxjs/operators';

describe('CDS: project Store', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: APP_BASE_HREF, useValue: '/'},
            ],
            imports: [
                AppModule,
                RouterModule,
                HttpClientTestingModule
            ]
        });

    });

    it('should Get Projects', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

        let projects = new Array<Project>();
        projects.push(createProject('key1', 'myProject'));
        projects.push(createProject('key2', 'myProject2'));
        projects.push(createProject('key3', 'myProject3'));
        projects.push(createProject('key4', 'myProject4'));

        // Get projects from HTTP Call
        projectStore.getProjectsList().pipe(first()).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project'
        })).flush(projects);

        projectStore.getProjectsList().pipe(first()).subscribe(pdata => {
            expect(pdata.size).toBe(4, 'Wrong number of project. Must be 4.');
        });

        http.verify();
    }));

    it('should create a Project', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

        let project = createProject('key1', 'myProject');
        let project2 = createProject('key2', 'myProject2');

        // Create project
        projectStore.createProject(createProject('key1', 'myProject')).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project'
        })).flush(project);

        projectStore.createProject(createProject('key2', 'myProject2')).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project'
        })).flush(project2);


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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(project);
        let checkedSingleProject = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            expect(projs.get('key1').name).toBe('myProject', 'Wrong project name. Must be myProject');
            checkedSingleProject = true;
        });
        expect(checkedSingleProject).toBeTruthy('Need to get project key1');


        // Delete project
        projectStore.deleteProject('key1').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(null);

        let checkedDeleteProject = false;
        projectStore.getProjects('key1').subscribe(() => {
            checkedDeleteProject = true;
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(null);

        http.verify();
    }));

    it('should update application in Project', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

        let projectMock = createProject('key1', 'myProject');
        projectMock.applications = new Array<Application>();
        projectMock.applications.push(createApplication('app1'));
        projectMock.applications.push(createApplication('app2'));
        projectMock.applications.push(createApplication('app3'));


        // Create project
        let p = createProject('key1', 'myProject');
        projectStore.createProject(p).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project'
        })).flush(projectMock);

        let projectChecked = false;
        projectStore.getProjects('key1').subscribe(projs => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectMock);
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            expect(projs.get('key1').applications.length).toBe(3, 'Wrong number of applications. Must be 3.');
            projectChecked = true;
        });
        expect(projectChecked).toBeTruthy();

        projectStore.updateApplicationName('key1', 'app2', 'appupdated');

        let projectAppChecked = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            expect(projs.get('key1').applications.length).toBe(3, 'Wrong number of applications. Must be 3.');
            expect(projs.get('key1').applications[1].name).toBe('appupdated');
            projectAppChecked = true;
        });
        expect(projectAppChecked).toBeTruthy();

        http.verify();
    }));

    it('should update Project', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

        let projectAdd = createProject('key1', 'myProject');
        let projectUp = createProject('key1', 'myProjectUpdate1');
        let projectUp2 = createProject('key1', 'myProjectUpdate2');

        // Create project
        let p = createProject('key1', 'myProject');
        projectStore.createProject(p).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project'
        })).flush(projectAdd);

        // Update
        p.name = 'myProjectUpdate1';
        projectStore.updateProject(p).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectUp);

        // check cache for nav
        let checkedNav = false;
        projectStore.getProjectsList().pipe(first()).subscribe(projects => {
            expect(projects.size).toBe(1, 'Wrong number of project. Must be 1.');
            expect(projects.get(0).name).toBe('myProjectUpdate1', 'Wrong name of project. Must be myProjectUpdate1');
            checkedNav = true;
        });
        expect(checkedNav).toBeTruthy('Need at least 2 projects in nav');

        // check get project: first time ( get from http call)
        projectStore.getProjects('key1').subscribe(projs => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectUp);
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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectUp2);

        // check get project: second time (get from cache
        let checkedSingleProjectTwice = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            expect(projs.get('key1').name).toBe('myProjectUpdate2', 'Wrong project name. Must be myProjectUpdate2');
            checkedSingleProjectTwice = true;
        });
        expect(checkedSingleProjectTwice).toBeTruthy('Need to get project key1 twice');

        http.verify();

    }));

    it('should add a repomananger + validate repo manager + delete repomanager', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

        let projectAdd = createProject('key1', 'myProject');

        let projectRepoAdd: any = {
            'project_key': 'key1',
            'last_modified': '123',
            'repositories_manager': 'repoName',
            'url': 'ff/.c',
            'request_token': 'mytoken'
        };

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectAdd);


        // Add repo
        projectStore.connectRepoManager('key1', 'repoName').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/repositories_manager/repoName/authorize'
        })).flush(projectRepoAdd);

        let addRepoCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addRepoCheck = true;
            expect(projs.get('key1').last_modified).toBe('123', 'Project must have been updated');
        });
        expect(addRepoCheck).toBeTruthy('Must check project update');

        // validate repo
        projectStore.verificationCallBackRepoManager('key1', 'repoName', 'token', 'code').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/repositories_manager/repoName/authorize/callback'
        })).flush(projectValidation);

        let validationRepoCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            validationRepoCheck = true;
            expect(projs.get('key1').last_modified).toBe('456');
            expect(projs.get('key1').vcs_servers.length).toBe(1);
        });
        expect(validationRepoCheck).toBeTruthy('Must check project update');

        projectStore.disconnectRepoManager('key1', 'repoName').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/repositories_manager/repoName'
        })).flush(projectRepoDel);

        let deleteRepoCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteRepoCheck = true;
            expect(projs.get('key1').last_modified).toBe('789');
            expect(projs.get('key1').vcs_servers.length).toBe(0);
        });
        expect(deleteRepoCheck).toBeTruthy('Must check project update');

        http.verify();
    }));

    it('should add/update/delete a variable', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectAdd);

        // Add variable
        let v: Variable = new Variable();
        v.name = 'var1';
        v.type = 'string';
        v.value = 'value1';
        v.description = 'myDescription';
        projectStore.addProjectVariable('key1', v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/variable/var1'
        })).flush(projectAddVar);

        let addVariableCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addVariableCheck = true;
            expect(projs.get('key1').last_modified).toBe('123', 'Project must have been updated');
            expect(projs.get('key1').variables.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').variables[0].name).toBe('myvar');
        });
        expect(addVariableCheck).toBeTruthy('Must check project update');


        // update variable
        projectStore.updateProjectVariable('key1', v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/variable/var1'
        })).flush(projectUpVar.variables[0]);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/variable/var1'
        })).flush(projectDelVar);

        let deleteVariableCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteVariableCheck = true;
            expect(projs.get('key1').variables.length).toBe(0);
        });
        expect(deleteVariableCheck).toBeTruthy('Must check project update');

        http.verify();
    }));

    it('should add/update/delete a permission', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectAdd);


        // Add groupPermission
        let gp: GroupPermission = new GroupPermission();
        gp.group = new Group();
        gp.group.id = 1;
        gp.group.name = 'mygroup';
        gp.permission = 7;
        projectStore.addProjectPermission('key1', gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/group'
        })).flush(projectAddGrp.groups);

        let addPermissionCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addPermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(1, 'Project must have 1 group');
            expect(projs.get('key1').groups[0].permission).toBe(7);
        });
        expect(addPermissionCheck).toBeTruthy('Must check project update');


        // update permission
        projectStore.updateProjectPermission('key1', gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/group/mygroup'
        })).flush(projectUpGrp.groups[0]);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/group/mygroup'
        })).flush(projectDelGrp.groups);

        let deletePermissionCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deletePermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(0);
        });
        expect(deletePermissionCheck).toBeTruthy('Must check project update');

        http.verify();
    }));

    it('should add/update/delete an environment', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectAdd);


        // Add evn
        let env: Environment = new Environment();
        env.name = 'Production';
        projectStore.addProjectEnvironment('key1', env).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment'
        })).flush(projectAddEnv);

        let addEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe('123', 'Project must have been updated');
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').environments[0].name).toBe('Production');
        });
        expect(addEnvCheck).toBeTruthy('Must check project update');


        // update env
        projectStore.renameProjectEnvironment('key1', env.name, env).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/Production'
        })).flush(projectUpEnv);

        let renameEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            renameEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe('456');
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].name).toBe('PreProduction');
        });
        expect(renameEnvCheck).toBeTruthy('Must check project update');

        // Delete env
        env.name = 'PreProduction';
        projectStore.deleteProjectEnvironment('key1', env).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/PreProduction'
        })).flush(projectDelEnv);

        let deleteEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe('789');
            expect(projs.get('key1').environments.length).toBe(0);
        });
        expect(deleteEnvCheck).toBeTruthy('Must check project delete env');

        http.verify();
    }));

    it('should add/update/delete an environment variable', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectAdd);


        // Add env variable
        let v: Variable = new Variable();
        v.name = 'foo';
        projectStore.addEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/prod/variable/foo'
        })).flush(projectAddEnvVar);

        let addEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe('123', 'Project must have been updated');
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').environments[0].variables.length).toBe(1);
            expect(projs.get('key1').environments[0].variables[0].name).toBe('foo');
        });
        expect(addEnvCheck).toBeTruthy('Must check project update');


        // update var
        projectStore.updateEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/prod/variable/foo'
        })).flush(projectUpEnvVar);

        let renameVarEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            renameVarEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe('456');
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].variables.length).toBe(1);
            expect(projs.get('key1').environments[0].variables[0].name).toBe('fooUpdated');
        });
        expect(renameVarEnvCheck).toBeTruthy('Must check project update');

        // Delete var
        v.name = 'fooUpdated';
        projectStore.removeEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/prod/variable/fooUpdated'
        })).flush(projectDelEnvVar);

        let deleteVarCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deleteVarCheck = true;
            expect(projs.get('key1').last_modified).toBe('789');
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].variables.length).toBe(0);
        });
        expect(deleteVarCheck).toBeTruthy('Must check project delete var');

        http.verify();
    }));

    it('should add/update/delete an environment permission', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const http = TestBed.get(HttpTestingController);

        let grp = new Group();
        grp.name = 'grp1';

        let projectAdd = createProject('key1', 'myProject');
        projectAdd.environments = new Array<Environment>();
        let envAdd = new Environment();
        envAdd.name = 'prod';
        projectAdd.environments.push(envAdd);

        let projectAddEnvGroup = createProject('key1', 'myProject');
        projectAdd.last_modified = '123';
        projectAddEnvGroup.environments = new Array<Environment>();
        let envAddGrp = new Environment();
        envAddGrp.name = 'prod';
        envAddGrp.groups = new Array<GroupPermission>();
        let gpAdd = new GroupPermission();
        gpAdd.group = grp;
        gpAdd.permission = 7;
        envAddGrp.groups.push(gpAdd);
        projectAddEnvGroup.environments.push(envAddGrp);

        let projectUpEnvGroup = createProject('key1', 'myProject');
        projectUpEnvGroup.last_modified = '456';
        projectUpEnvGroup.environments = new Array<Environment>();
        let envUpGrp = new Environment();
        envUpGrp.name = 'prod';
        envUpGrp.groups = new Array<GroupPermission>();
        let gpUp = new GroupPermission();
        gpUp.group = grp;
        gpUp.permission = 4;
        envUpGrp.groups.push(gpUp);
        projectUpEnvGroup.environments.push(envUpGrp);

        let projectDelEnvGrp = createProject('key1', 'myProject');
        projectDelEnvGrp.last_modified = '789';
        projectDelEnvGrp.environments = new Array<Environment>();
        let envDel = new Environment();
        envDel.name = 'prod';
        projectDelEnvGrp.environments.push(envDel);

        // Get project in cache
        projectStore.getProjects('key1').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1'
        })).flush(projectAdd);

        // Add env permission
        let gpA: Array<GroupPermission> = new Array<GroupPermission>();
        let gp = new GroupPermission();
        gp.permission = 7;
        gp.group = new Group();
        gp.group.name = 'grp1';
        gpA.push(gp);
        projectStore.addEnvironmentPermission('key1', 'prod', gpA).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/prod/groups'
        })).flush(projectAddEnvGroup.environments[0]);

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
        projectStore.updateEnvironmentPermission('key1', 'prod', gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/prod/group/grp1'
        })).flush(projectUpEnvGroup.environments[0]);

        let renameVarEnvCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            renameVarEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].groups.length).toBe(1);
            expect(projs.get('key1').environments[0].groups[0].permission).toBe(4);
        });
        expect(renameVarEnvCheck).toBeTruthy('Must check env update');

        // Delete gp
        projectStore.removeEnvironmentPermission('key1', 'prod', gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/environment/prod/group/grp1'
        })).flush(null);

        let deletePermCheck = false;
        projectStore.getProjects('key1').pipe(first()).subscribe(projs => {
            deletePermCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].groups.length).toBe(0);
        });
        expect(deletePermCheck).toBeTruthy('Must check env delete perm');

        http.verify();
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
});
