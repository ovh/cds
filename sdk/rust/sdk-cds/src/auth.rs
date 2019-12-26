#[derive(Serialize, Deserialize)]
pub struct AuthClaims {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "aud")]
    pub audience: Option<String>,
    #[serde(rename = "exp")]
    pub expires_at: i64,
    pub jti: String,
    #[serde(rename = "iat")]
    pub issued_at: i64,
    #[serde(rename = "iss")]
    pub issuer: String,
    #[serde(rename = "nbf")]
    pub not_before: Option<i64>,
    #[serde(rename = "sub")]
    pub subject: String,
}
