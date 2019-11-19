import config from '../config';
import LoginPage from '../pages/login'
import NavbarPage from '../pages/navabar';
import ProjectCreatePage from '../pages/project-create';
import ProjectPage from '../pages/project';
import WorkflowCreatePage from '../pages/workflow-create';
import WorkflowPage from '../pages/workflow';
import { Selector } from 'testcafe';

// DATA
const projectName = 'CDSE2';
const projectKey = 'CDSE2';
const group = 'aa';
const workflowName = 'myFirstWorkflow';
const pipName = 'myFirstPipeline';
const appName = 'myFirstApplication';

const loginPage = new LoginPage();
const navbarPage = new NavbarPage();
const projectCreatePage = new ProjectCreatePage();
const projectPage = new ProjectPage(projectKey);
const workflowCreatePage = new WorkflowCreatePage(projectKey);
const workflowPage = new WorkflowPage(projectKey, workflowName);

fixture('workflow-create-and-run')
    .meta({
        severity: 'critical',
        priority: 'high',
        scope: 'workflow'
    })
    .beforeEach(async(t) => {
        await t.maximizeWindow();
        await loginPage.login(config.username, config.password);
    });

test('workflow-create-and-run', async (t) => {
    await navbarPage.clickCreateProject();
    await projectCreatePage.createProject(projectName, projectKey, group);
    await projectPage.clickCreateWorkflow();
    await workflowCreatePage.createWorkflow(workflowName, projectKey, pipName, appName);

    await workflowPage.addFork('div.node-wrapper');
    await workflowPage.addPipeline('div.node-wrapper', 1, pipName);
    await workflowPage.addPipeline('div.node-wrapper', 1, pipName);
    await workflowPage.addJoin('div.node-wrapper', 2);
    await workflowPage.linkJoin('div.node-wrapper', 3, 0);
    await workflowPage.addPipeline('div.node-wrapper', 4, pipName);

    await workflowPage.runWorkflow();
    await t.expect('.runs .item.pointing.success').ok();

});
