
use actix_web::{HttpRequest, HttpResponse};
use chrono::prelude::*;
use sdk_cds::models::{MonitoringStatus, StatusLine};

pub fn status_handler(_req: HttpRequest) -> HttpResponse {
    let status = MonitoringStatus {
        now: Some(Utc::now()),
        lines: Some(vec![StatusLine {
            component: String::from("Global"),
            status: String::from("OK"),
            value: String::default(),
            _type: String::default(),
        }]),
    };

    HttpResponse::Ok().json(status)
}