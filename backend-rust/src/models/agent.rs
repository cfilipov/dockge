use serde::{Deserialize, Serialize};
use sqlx::SqlitePool;

#[derive(Debug, Clone, sqlx::FromRow, Serialize, Deserialize)]
pub struct Agent {
    pub id: i64,
    pub url: String,
    pub username: String,
    pub password: String,
    pub name: Option<String>,
    pub active: bool,
}

impl Agent {
    /// Derive the endpoint (host:port) from the URL
    pub fn endpoint(&self) -> String {
        // Strip protocol prefix and path, keep host:port
        self.url
            .strip_prefix("https://")
            .or_else(|| self.url.strip_prefix("http://"))
            .unwrap_or(&self.url)
            .split('/')
            .next()
            .unwrap_or(&self.url)
            .to_string()
    }

    pub async fn find_all(pool: &SqlitePool) -> Result<Vec<Agent>, sqlx::Error> {
        sqlx::query_as::<_, Agent>(
            "SELECT id, url, username, password, name, active FROM agent WHERE active = 1"
        )
        .fetch_all(pool)
        .await
    }

    pub async fn find_by_url(pool: &SqlitePool, url: &str) -> Result<Option<Agent>, sqlx::Error> {
        sqlx::query_as::<_, Agent>(
            "SELECT id, url, username, password, name, active FROM agent WHERE url = ?"
        )
        .bind(url)
        .fetch_optional(pool)
        .await
    }

    pub async fn create(
        pool: &SqlitePool,
        url: &str,
        username: &str,
        password: &str,
        name: &str,
    ) -> Result<i64, sqlx::Error> {
        let result = sqlx::query(
            "INSERT INTO agent (url, username, password, name) VALUES (?, ?, ?, ?)"
        )
        .bind(url)
        .bind(username)
        .bind(password)
        .bind(name)
        .execute(pool)
        .await?;
        Ok(result.last_insert_rowid())
    }

    pub async fn delete(pool: &SqlitePool, url: &str) -> Result<bool, sqlx::Error> {
        let result = sqlx::query("DELETE FROM agent WHERE url = ?")
            .bind(url)
            .execute(pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    pub async fn update_name(pool: &SqlitePool, url: &str, name: &str) -> Result<bool, sqlx::Error> {
        let result = sqlx::query("UPDATE agent SET name = ? WHERE url = ?")
            .bind(name)
            .bind(url)
            .execute(pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }
}
