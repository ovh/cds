use actix::prelude::*;
use actix_web::{App, web, HttpServer};
use actix_web::{http, middleware};

use badge::handlers::badge_handler;
use database::DbExecutor;
// use middlewares::auth::AuthMiddleware;
// use run::handlers::run_handler;
use status::handlers::status_handler;

#[derive(Clone)]
pub struct WebState {
    pub db: Addr<DbExecutor>,
    pub hash: String,
}

pub fn http_server(state: WebState, http_bind: String, http_port: String) {
    use actix_web::middleware::cors::Cors;
    HttpServer::new(move || {
        App::new()
            .data(state.clone())
            .wrap(middleware::Logger::default())
            .wrap(Cors::new() // <- Construct CORS middleware builder
                    .allowed_methods(vec!["GET", "POST", "OPTION"])
                    .allowed_headers(vec![http::header::AUTHORIZATION, http::header::ACCEPT])
                    .allowed_header(http::header::CONTENT_TYPE)
                    .max_age(3600)
            ).service(web::resource("/{project}/{workflow}/badge.svg").to_async(badge_handler))
            .service(web::resource("/mon/status").to(status_handler))
                    
        // App::with_state()
            // .middleware(middleware::Logger::default())
            // .configure(|app| {
            //     Cors::new() // <- Construct CORS middleware builder
            //         .allowed_methods(vec!["GET", "POST", "OPTION"])
            //         .allowed_headers(vec![http::header::AUTHORIZATION, http::header::ACCEPT])
            //         .allowed_header(http::header::CONTENT_TYPE)
            //         .max_age(3600)
            //         .resource("/mon/status", |r| {
            //             r.method(http::Method::GET).f(status_handler)
            //         })
            //         .resource("/{project}/{workflow}/badge.svg", |r| {
            //             r.method(http::Method::GET).f(badge_handler)
            //         })
            //         .register()
            // })
    })
    .bind(format!("{}:{}", http_bind, http_port))
    .unwrap()
    .run();
}
