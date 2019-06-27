
use actix_web::error;
use actix_web::http::HeaderMap;
use actix_web::{web, Error, HttpRequest, HttpResponse};
use badge_gen::{Badge, BadgeOptions};
use futures::Future;
use std::collections::HashMap;

use crate::models::StatusEnum;
use crate::run::QueryRun;
use crate::web::WebState;

const GREEN: &str = "#21BA45";
const RED: &str = "#FF4F60";
const BLUE: &str = "#4fa3e3";

pub fn badge_handler(
    state: web::Data<WebState>,
    query: web::Query<HashMap<String, String>>,
    req: HttpRequest,
) -> impl Future<Item = HttpResponse, Error = Error> {
    let project_key = req.match_info().get("project").unwrap_or_default();
    let workflow_name = req.match_info().get("workflow").unwrap_or_default();
    let branch = query
        .get("branch")
        .map(|q| q.to_string())
        .or_else(|| get_branch_from_referer(req.headers()));

    let query = QueryRun {
        project_key: project_key.to_string(),
        workflow_name: workflow_name.to_string(),
        branch,
    };

    state.db_actor.send(query).from_err().and_then(move |res| {
        let run = res?;
        let color = match StatusEnum::from(run.status.clone()) {
            StatusEnum::Success => GREEN.to_string(),
            StatusEnum::Building | StatusEnum::Waiting | StatusEnum::Checking => String::from(BLUE),
            StatusEnum::Fail | StatusEnum::Stopped => RED.to_string(),
            _ => "grey".to_string(),
        };

        let opts = BadgeOptions {
            subject: String::from("CDS"),
            status: run.status,
            color,
        };
        let badge = Badge::new(opts).map_err(error::ErrorBadRequest)?.to_svg();

        Ok(HttpResponse::Ok().content_type("image/svg+xml").body(badge))
    })
}

fn get_branch_from_referer(headers: &HeaderMap) -> Option<String> {
    let mut branch = None;
    if let Some(ref referer_value) = headers.get("Referer") {
        let referer_value_str = referer_value.to_str().unwrap();
        if let Some(tree_index) = referer_value_str.find("/tree/") {
            branch = Some(referer_value_str[tree_index + 6..].to_string());
        } else if let Some(src_index) = referer_value_str.find("/src/") {
            branch = Some(
                referer_value_str[src_index + 5..]
                    .trim_end_matches('/')
                    .to_string(),
            );
        }
    }

    branch
}
