import { ClientFunction, Selector, t } from 'testcafe';
import config from '../config';


export default class LoginPage {

    url: string;
    userNameInput: Selector;
    passwordInput: Selector;
    loginButton: Selector;

    getLocation = ClientFunction(() => document.location.href);

    constructor(){
        this.url = config.baseUrl + '/account/login';
        this.userNameInput = Selector('input[name=username]');
        this.passwordInput = Selector('input[name=password]');
        this.loginButton = Selector('.ui.green.button');
    }

    async login() {
        await t
            .typeText(this.userNameInput, 'aa')
            .typeText(this.passwordInput, '46738385e8d13d2ce4412e5aebb189c0fa65cd4745c182777bc4f1bf408d25bf07c4ad9bea4454fd5927c5beefd1138d45387697b2e94a12ecd5869148dfafc5')
            .click(this.loginButton)
            .expect(this.getLocation()).eql(config.baseUrl + '/home');
    }
}
