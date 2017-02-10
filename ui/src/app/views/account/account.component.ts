import {AuthentificationStore} from '../../service/auth/authentification.store';

export class AccountComponent {
    constructor(_authStore: AuthentificationStore) {
        _authStore.removeUser();
    }
}
