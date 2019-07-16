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

    async login(user: string, password: string) {
        await t
            .navigateTo(this.url)
            .typeText(this.userNameInput, user)
            .typeText(this.passwordInput, password)
            .click(this.loginButton)
            .expect(this.getLocation()).eql(config.baseUrl + '/home');
    }
}
