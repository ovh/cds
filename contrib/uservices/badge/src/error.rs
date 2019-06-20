use actix_web::{HttpResponse, ResponseError};
use failure::Error;

#[derive(Fail, Debug)]
pub enum BadgeError {
    #[fail(display = "Invalid parameter")]
    InvalidParameter,
    #[fail(display = "No run available")]
    NoRunAvailable,
    #[fail(display = "Unauthorised")]
    Unauthorised,
    #[fail(display = "Unexpected error")]
    UnexpectedError { cause: Error },
}

impl ResponseError for BadgeError {
    fn error_response(&self) -> HttpResponse {
        match *self {
            BadgeError::Unauthorised => HttpResponse::Unauthorized().json("Unauthorised"),
            BadgeError::NoRunAvailable => HttpResponse::NotFound().json("Not run found"),
            BadgeError::InvalidParameter => HttpResponse::BadRequest().json("Invalid parameter"),
            _ => HttpResponse::InternalServerError().json("Unexpected error"),
        }
    }
}
