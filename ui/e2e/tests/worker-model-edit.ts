import config from '../config';
import LoginPage from '../pages/login'
import NavbarPage from '../pages/navabar';
import WorkerModelPage from '../pages/worker-model';
import WorkerModelEditPage from '../pages/worker-model-create';

const MODEL_NAME = 'cds-e2e-model';
const GROUP_NAME = 'shared.infra';
const IMAGE = 'debian:9';

const loginPage = new LoginPage();
const navbarPage = new NavbarPage();
const workerModelPage = new WorkerModelPage();
const workerModelEditPage = new WorkerModelEditPage();

fixture('worker-model-create')
    .meta({
        severity: 'critical',
        priority: 'high',
        scope: 'workflow'
    })
    .beforeEach(async(t) => {
        await t.maximizeWindow();
        await loginPage.login(config.username, config.password);
    });

test('worker-model-create', async (t) => {
    await navbarPage.clickWorkerModel();
    await workerModelPage.clickAddButton();
    await workerModelEditPage.createDockerModel(MODEL_NAME, GROUP_NAME, IMAGE);
});
