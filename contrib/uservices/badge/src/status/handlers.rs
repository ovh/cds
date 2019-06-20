use chrono::prelude::*;
use actix_web::{HttpRequest, HttpResponse};
use sdk_cds::models::{MonitoringStatus, StatusLine};

use crate::web::WebState;

pub fn status_handler(req: &HttpRequest<WebState>) -> HttpResponse {
    let status = MonitoringStatus{
        now: Some(Utc::now()),
        lines: Some(
            vec![
                StatusLine{
                    component: String::from("Global"),
                    status: String::from("OK"),
                    value: String::default(),
                    _type: String::default(),
                }
            ]
        )
    };

    HttpResponse::Ok().json(status)
}