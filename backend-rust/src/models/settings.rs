use serde_json::Value;
use sqlx::SqlitePool;
use std::collections::HashMap;
use std::sync::LazyLock;
use tokio::sync::RwLock;
use std::time::Instant;

use crate::error::AppResult;

struct CacheEntry {
    value: Value,
    timestamp: Instant,
}

static SETTINGS_CACHE: LazyLock<RwLock<HashMap<String, CacheEntry>>> =
    LazyLock::new(|| RwLock::new(HashMap::new()));

const CACHE_TTL_SECS: u64 = 60;

/// Get a single setting value by key
pub async fn get(pool: &SqlitePool, key: &str) -> AppResult<Option<Value>> {
    // Check cache first
    {
        let cache = SETTINGS_CACHE.read().await;
        if let Some(entry) = cache.get(key) {
            if entry.timestamp.elapsed().as_secs() < CACHE_TTL_SECS {
                return Ok(Some(entry.value.clone()));
            }
        }
    }

    let row: Option<(Option<String>,)> = sqlx::query_as(
        "SELECT value FROM setting WHERE key = ?"
    )
    .bind(key)
    .fetch_optional(pool)
    .await?;

    if let Some((Some(value_str),)) = row {
        let value: Value = serde_json::from_str(&value_str).unwrap_or(Value::String(value_str.clone()));

        // Cache it
        let mut cache = SETTINGS_CACHE.write().await;
        cache.insert(key.to_string(), CacheEntry {
            value: value.clone(),
            timestamp: Instant::now(),
        });

        Ok(Some(value))
    } else {
        Ok(None)
    }
}

/// Set a single setting value
pub async fn set(pool: &SqlitePool, key: &str, value: &Value, setting_type: Option<&str>) -> AppResult<()> {
    let value_str = serde_json::to_string(value)?;
    let type_str = setting_type.unwrap_or("");

    // Upsert
    let existing: Option<(i64,)> = sqlx::query_as(
        "SELECT id FROM setting WHERE key = ?"
    )
    .bind(key)
    .fetch_optional(pool)
    .await?;

    if let Some((id,)) = existing {
        sqlx::query("UPDATE setting SET value = ?, type = ? WHERE id = ?")
            .bind(&value_str)
            .bind(type_str)
            .bind(id)
            .execute(pool)
            .await?;
    } else {
        sqlx::query("INSERT INTO setting (key, value, type) VALUES (?, ?, ?)")
            .bind(key)
            .bind(&value_str)
            .bind(type_str)
            .execute(pool)
            .await?;
    }

    // Invalidate cache
    let mut cache = SETTINGS_CACHE.write().await;
    cache.remove(key);

    Ok(())
}

/// Get all settings of a given type as a JSON object
pub async fn get_settings(pool: &SqlitePool, setting_type: &str) -> AppResult<serde_json::Map<String, Value>> {
    let rows: Vec<(String, Option<String>)> = sqlx::query_as(
        "SELECT key, value FROM setting WHERE type = ?"
    )
    .bind(setting_type)
    .fetch_all(pool)
    .await?;

    let mut result = serde_json::Map::new();
    for (key, value_opt) in rows {
        if let Some(value_str) = value_opt {
            let value: Value = serde_json::from_str(&value_str)
                .unwrap_or(Value::String(value_str));
            result.insert(key, value);
        }
    }

    Ok(result)
}

/// Set multiple settings of a given type
pub async fn set_settings(pool: &SqlitePool, setting_type: &str, data: &serde_json::Map<String, Value>) -> AppResult<()> {
    for (key, value) in data {
        let value_str = serde_json::to_string(value)?;

        let existing: Option<(i64, Option<String>)> = sqlx::query_as(
            "SELECT id, type FROM setting WHERE key = ?"
        )
        .bind(key)
        .fetch_optional(pool)
        .await?;

        if let Some((id, existing_type)) = existing {
            // Only update if the type matches
            if existing_type.as_deref() == Some(setting_type) || existing_type.is_none() {
                sqlx::query("UPDATE setting SET value = ?, type = ? WHERE id = ?")
                    .bind(&value_str)
                    .bind(setting_type)
                    .bind(id)
                    .execute(pool)
                    .await?;
            }
        } else {
            sqlx::query("INSERT INTO setting (key, value, type) VALUES (?, ?, ?)")
                .bind(key)
                .bind(&value_str)
                .bind(setting_type)
                .execute(pool)
                .await?;
        }
    }

    // Invalidate cache for all changed keys
    let mut cache = SETTINGS_CACHE.write().await;
    for key in data.keys() {
        cache.remove(key);
    }

    Ok(())
}

/// Clear all cached settings
pub async fn clear_cache() {
    let mut cache = SETTINGS_CACHE.write().await;
    cache.clear();
}
