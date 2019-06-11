
use actix::Addr;
use actix_web::{http, middleware};
use actix_web::{web, App, HttpServer};
use diesel::r2d2::{self, ConnectionManager};
use diesel::PgConnection;


use crate::database::DbExecutor;
use badge::handlers::badge_handler;
// use run::handlers::run_handler;
use status::handlers::status_handler;

pub type Pool = r2d2::Pool<ConnectionManager<PgConnection>>;

#[derive(Clone)]
pub struct WebState {
    pub db: Pool,
    pub db_actor: Addr<DbExecutor>,
}

pub fn http_server(state: WebState, http_bind: String, http_port: String) {
    use actix_web::middleware::cors::Cors;
    HttpServer::new(move || {
        App::new()
            .data(state.clone())
            .wrap(middleware::Logger::default())
            .wrap(
                Cors::new() // <- Construct CORS middleware builder
                    .allowed_methods(vec!["GET", "POST", "OPTION"])
                    .allowed_headers(vec![http::header::AUTHORIZATION, http::header::ACCEPT])
                    .allowed_header(http::header::CONTENT_TYPE)
                    .max_age(3600),
            )
            .service(web::resource("/{project}/{workflow}/badge.svg").to_async(badge_handler))
            .service(web::resource("/mon/status").to(status_handler))
    })
    .bind(format!("{}:{}", http_bind, http_port))
    .unwrap()
    .start();
}
