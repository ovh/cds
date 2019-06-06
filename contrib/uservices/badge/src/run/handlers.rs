use actix_web::{HttpRequest, HttpResponse, FutureResponse, AsyncResponder, Json};
use futures::Future;

use crate::run::CreateRun;
use crate::models::Run;
use crate::web::WebState;

pub fn run_handler((run_payload, req): (Json<Run>, HttpRequest<WebState>)) -> FutureResponse<HttpResponse> {
    req.state().db.send(CreateRun{
            run: run_payload.into_inner(),
        })
        .from_err()
        .and_then(|res| {
            Ok(HttpResponse::Ok().content_type("application/json").json(res?))
        }).responder()
}