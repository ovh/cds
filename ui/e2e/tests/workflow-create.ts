import config from '../config';
import LoginPage from '../pages/login'
import NavbarPage from '../pages/navabar';
import ProjectCreatePage from '../pages/project-create';
import ProjectPage from '../pages/project';
import WorkflowCreatePage from '../pages/workflow-create';

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

fixture('workflow-create')
    .meta({
        severity: 'critical',
        priority: 'high',
        scope: 'workflow'
    })
    .beforeEach(async(t) => {
        await t
            .maximizeWindow()
            .navigateTo(config.baseUrl)
    });

test('workflow-create', async (t) => {
    await loginPage.login();
    await navbarPage.clickCreateProject();
    await projectCreatePage.createProject(projectName, projectKey, group);
    await projectPage.clickCreateWorkflow();
    await workflowCreatePage.createWorkflow(workflowName, projectKey, pipName, appName);
});
