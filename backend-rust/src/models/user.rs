use jsonwebtoken::{encode, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
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
    pub exp: usize,
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
            exp,
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

/// SHAKE256-like hash using SHA256 truncated (simplified)
/// The Node.js backend uses shake256 from crypto, we approximate with SHA256
pub fn shake256_hex(input: &str, length: usize) -> String {
    let mut hasher = Sha256::new();
    hasher.update(input.as_bytes());
    let result = hasher.finalize();
    let hex_str = hex::encode(result);
    hex_str[..std::cmp::min(length * 2, hex_str.len())].to_string()
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
