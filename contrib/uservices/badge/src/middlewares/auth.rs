use actix_web::middleware::{Middleware, Started};
use actix_web::HttpRequest;

use crate::error::BadgeError;
use crate::web::WebState;

// create a middleware
pub struct AuthMiddleware;
impl Middleware<WebState> for AuthMiddleware {
    fn start(&self, req: &HttpRequest<WebState>) -> actix_web::Result<Started> {
        // don't validate CORS pre-flight requests
        if req.method() == "OPTIONS" {
            return Ok(Started::Done);
        }

        let token = req
            .headers()
            .get("X_AUTH_HEADER")
            .map(|value| value.to_str().ok())
            .ok_or(BadgeError::Unauthorised)?;

        match token {
            Some(t) if t != "" => {
                if req.state().hash != t {
                    return Err(BadgeError::Unauthorised.into());
                }
                Ok(Started::Done)
            }
            _ => Err(BadgeError::Unauthorised.into()),
        }
    }
}
