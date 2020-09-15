import { ChangeDetectionStrategy, Component } from '@angular/core';
import { Toast } from 'angular2-toaster-sgu';

export class ToastHTTPErrorData {
    status: number;
    from: string;
    request_id: string;
    help: string;
}

@Component({
    selector: 'app-toast-http-error-component',
    templateUrl: './toast-http-error.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ToastHTTPErrorComponent {
    toast: Toast;
}
