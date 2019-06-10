use actix::prelude::*;
use actix_web::server::HttpServer;
use actix_web::App;
use actix_web::{http, middleware};

use badge::handlers::badge_handler;
use database::DbExecutor;
use middlewares::auth::AuthMiddleware;
use run::handlers::run_handler;
use status::handlers::status_handler;

#[derive(Clone)]
pub struct WebState {
    pub db: Addr<DbExecutor>,
    pub hash: String,
}

pub fn http_server(state: WebState, http_bind: String, http_port: String) {
    use actix_web::middleware::cors::Cors;
    HttpServer::new(move || {
        App::with_state(state.clone())
            .middleware(middleware::Logger::default())
            .configure(|app| {
                Cors::for_app(app) // <- Construct CORS middleware builder
                    .allowed_methods(vec!["GET", "POST", "OPTION"])
                    .allowed_headers(vec![http::header::AUTHORIZATION, http::header::ACCEPT])
                    .allowed_header(http::header::CONTENT_TYPE)
                    .max_age(3600)
                    .resource("/mon/status", |r| {
                        r.method(http::Method::GET).f(status_handler)
                    })
                    .resource("/{project}/{workflow}/badge.svg", |r| {
                        r.method(http::Method::GET).f(badge_handler)
                    })
                    .resource("/run", |r| {
                        r.middleware(AuthMiddleware);
                        r.method(http::Method::POST).with_async(run_handler);
                    })
                    .register()
            })
    })
    .bind(format!("{}:{}", http_bind, http_port))
    .unwrap()
    .start();
}
