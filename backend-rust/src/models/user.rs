use jsonwebtoken::{encode, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use sqlx::SqlitePool;

use crate::error::{AppError, AppResult};

#[derive(Debug, Clone, sqlx::FromRow)]
#[allow(dead_code)]
pub struct User {
    pub id: i64,
    pub username: String,
    pub password: Option<String>,
    pub active: bool,
    pub timezone: Option<String>,
    pub twofa_secret: Option<String>,
    pub twofa_status: bool,
    pub twofa_last_token: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct JwtClaims {
    pub username: String,
    pub h: String, // hash of password for invalidation on password change
    #[serde(default)]
    pub exp: Option<usize>,
}

impl User {
    pub async fn find_by_username(pool: &SqlitePool, username: &str) -> AppResult<Option<User>> {
        let user = sqlx::query_as::<_, User>(
            "SELECT id, username, password, active, timezone, twofa_secret, twofa_status, twofa_last_token FROM user WHERE username = ? AND active = 1"
        )
        .bind(username)
        .fetch_optional(pool)
        .await?;
        Ok(user)
    }

    pub async fn find_by_id(pool: &SqlitePool, id: i64) -> AppResult<Option<User>> {
        let user = sqlx::query_as::<_, User>(
            "SELECT id, username, password, active, timezone, twofa_secret, twofa_status, twofa_last_token FROM user WHERE id = ? AND active = 1"
        )
        .bind(id)
        .fetch_optional(pool)
        .await?;
        Ok(user)
    }

    pub async fn find_first(pool: &SqlitePool) -> AppResult<Option<User>> {
        let user = sqlx::query_as::<_, User>(
            "SELECT id, username, password, active, timezone, twofa_secret, twofa_status, twofa_last_token FROM user LIMIT 1"
        )
        .fetch_optional(pool)
        .await?;
        Ok(user)
    }

    pub async fn count(pool: &SqlitePool) -> AppResult<i64> {
        let row: (i64,) = sqlx::query_as("SELECT COUNT(id) FROM user")
            .fetch_one(pool)
            .await?;
        Ok(row.0)
    }

    pub async fn create(pool: &SqlitePool, username: &str, password_hash: &str) -> AppResult<i64> {
        let result = sqlx::query(
            "INSERT INTO user (username, password) VALUES (?, ?)"
        )
        .bind(username)
        .bind(password_hash)
        .execute(pool)
        .await?;
        Ok(result.last_insert_rowid())
    }

    pub async fn update_password(pool: &SqlitePool, user_id: i64, password_hash: &str) -> AppResult<()> {
        sqlx::query("UPDATE user SET password = ? WHERE id = ?")
            .bind(password_hash)
            .bind(user_id)
            .execute(pool)
            .await?;
        Ok(())
    }

    pub fn create_jwt(&self, jwt_secret: &str) -> AppResult<String> {
        let password_hash = self.password.as_deref().unwrap_or("");
        let h = shake256_hex(password_hash, 16);

        let exp = chrono::Utc::now()
            .checked_add_signed(chrono::Duration::days(30))
            .unwrap()
            .timestamp() as usize;

        let claims = JwtClaims {
            username: self.username.clone(),
            h,
            exp: Some(exp),
        };

        encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(jwt_secret.as_bytes()),
        )
        .map_err(|e| AppError::Internal(format!("JWT encode error: {}", e)))
    }

    pub fn verify_password(&self, password: &str) -> bool {
        if let Some(ref stored) = self.password {
            verify_password_hash(password, stored)
        } else {
            false
        }
    }
}

/// Generate a bcrypt password hash
pub fn generate_password_hash(password: &str) -> String {
    bcrypt::hash(password, 10).expect("bcrypt hash failed")
}

/// Verify a password against a bcrypt hash
pub fn verify_password_hash(password: &str, hash: &str) -> bool {
    bcrypt::verify(password, hash).unwrap_or(false)
}

/// SHAKE256 hash matching Node.js crypto.createHash("shake256", { outputLength: len })
pub fn shake256_hex(input: &str, length: usize) -> String {
    if input.is_empty() {
        return String::new();
    }
    use sha3::Shake256;
    use sha3::digest::{Update, ExtendableOutput, XofReader};
    let mut hasher = Shake256::default();
    Update::update(&mut hasher, input.as_bytes());
    let mut reader = hasher.finalize_xof();
    let mut buf = vec![0u8; length];
    reader.read(&mut buf);
    hex::encode(buf)
}

/// Verify a TOTP token against a secret (RFC 6238)
/// Secret is expected as base32-encoded string (standard for authenticator apps).
/// Returns true if the token matches within ±1 time step.
pub fn verify_totp(token: &str, secret_b32: &str) -> bool {
    use data_encoding::BASE32_NOPAD;

    // Decode base32 secret (try with and without padding)
    let secret = BASE32_NOPAD
        .decode(secret_b32.trim().to_uppercase().as_bytes())
        .or_else(|_| data_encoding::BASE32.decode(secret_b32.trim().to_uppercase().as_bytes()))
        .unwrap_or_else(|_| secret_b32.as_bytes().to_vec()); // fallback: raw bytes

    let time_step = 30u64;
    let digits = 6u32;
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs();
    let counter = now / time_step;

    // Check ±1 time step window
    for offset in -1i64..=1 {
        let check_counter = (counter as i64 + offset) as u64;
        let code = generate_hotp(&secret, check_counter, digits);
        if code == token {
            return true;
        }
    }
    false
}

/// Generate an HOTP code (RFC 4226)
fn generate_hotp(secret: &[u8], counter: u64, digits: u32) -> String {
    use hmac::{Hmac, Mac};
    use sha1::Sha1;

    let mut mac = <Hmac<Sha1>>::new_from_slice(secret).expect("HMAC accepts any key length");
    mac.update(&counter.to_be_bytes());
    let result = mac.finalize().into_bytes();

    let offset = (result[19] & 0x0f) as usize;
    let code = ((result[offset] as u32 & 0x7f) << 24)
        | ((result[offset + 1] as u32) << 16)
        | ((result[offset + 2] as u32) << 8)
        | (result[offset + 3] as u32);

    let code = code % 10u32.pow(digits);
    format!("{:0>width$}", code, width = digits as usize)
}

impl User {
    pub async fn update_twofa_last_token(pool: &SqlitePool, user_id: i64, token: &str) -> AppResult<()> {
        sqlx::query("UPDATE user SET twofa_last_token = ? WHERE id = ?")
            .bind(token)
            .bind(user_id)
            .execute(pool)
            .await?;
        Ok(())
    }
}

/// Check if a password is strong enough
pub fn check_password_strength(password: &str) -> bool {
    if password.len() < 6 {
        return false;
    }
    let has_alpha = password.chars().any(|c| c.is_alphabetic());
    let has_digit = password.chars().any(|c| c.is_ascii_digit());
    has_alpha && has_digit
}

/// Generate a random secret string
pub fn gen_secret(length: usize) -> String {
    use rand::Rng;
    const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    let mut rng = rand::rng();
    (0..length)
        .map(|_| {
            let idx = rng.random_range(0..CHARSET.len());
            CHARSET[idx] as char
        })
        .collect()
}
