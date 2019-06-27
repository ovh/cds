
use actix::MailboxError;
use actix_web::{HttpResponse, ResponseError};
use diesel::result::Error as DieselError;
use failure::Error;
#[derive(Fail, Debug)]
pub enum BadgeError {
    #[fail(display = "Invalid parameter")]
    InvalidParameter,
    #[fail(display = "No run available")]
    NoRunAvailable,
    #[fail(display = "Unauthorised")]
    Unauthorised,
    #[fail(display = "Database Error")]
    DieselError { cause: DieselError },
    #[fail(display = "Mailbox actix Error")]
    MailboxError { cause: MailboxError },
    #[fail(display = "Unexpected error")]
    UnexpectedError { cause: Error },
}

impl ResponseError for BadgeError {
    fn error_response(&self) -> HttpResponse {
        match *self {
            BadgeError::Unauthorised => HttpResponse::Unauthorized().json("Unauthorised"),
            BadgeError::NoRunAvailable => HttpResponse::NotFound().json("Not run found"),
            BadgeError::InvalidParameter => HttpResponse::BadRequest().json("Invalid parameter"),
            BadgeError::DieselError{..} => HttpResponse::InternalServerError().json("Database error"),
            BadgeError::MailboxError{..} => HttpResponse::InternalServerError().json("Communication error"),
            _ => HttpResponse::InternalServerError().json("Unexpected error"),
        }
    }
}

impl From<DieselError> for BadgeError {
    fn from(err: DieselError) -> Self {
        BadgeError::DieselError { cause: err }
    }
}

impl From<MailboxError> for BadgeError {
    fn from(err: MailboxError) -> Self {
        BadgeError::MailboxError { cause: err }
    }
}
