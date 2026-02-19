use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePool, SqlitePoolOptions, SqliteSynchronous};
use std::path::Path;
use tracing::info;

pub async fn init_pool(db_path: &Path) -> Result<SqlitePool, sqlx::Error> {
    // Ensure parent directory exists
    if let Some(parent) = db_path.parent() {
        std::fs::create_dir_all(parent).ok();
    }

    let options = SqliteConnectOptions::new()
        .filename(db_path)
        .create_if_missing(true)
        .journal_mode(SqliteJournalMode::Wal)
        .synchronous(SqliteSynchronous::Normal)
        .pragma("cache_size", "-256")
        .pragma("auto_vacuum", "INCREMENTAL")
        .pragma("foreign_keys", "ON");

    let pool = SqlitePoolOptions::new()
        .max_connections(2)
        .connect_with(options)
        .await?;

    info!("Connected to SQLite database at {}", db_path.display());

    Ok(pool)
}

pub async fn run_migrations(pool: &SqlitePool) -> Result<(), sqlx::Error> {
    // Create tables if they don't exist (matching the Node.js knex migrations)
    sqlx::query(
        "CREATE TABLE IF NOT EXISTS setting (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL UNIQUE COLLATE NOCASE,
            value TEXT,
            type TEXT
        )"
    )
    .execute(pool)
    .await?;

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS user (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL UNIQUE COLLATE NOCASE,
            password TEXT,
            active INTEGER NOT NULL DEFAULT 1,
            timezone TEXT,
            twofa_secret TEXT,
            twofa_status INTEGER NOT NULL DEFAULT 0,
            twofa_last_token TEXT
        )"
    )
    .execute(pool)
    .await?;

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS agent (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            url TEXT NOT NULL UNIQUE,
            username TEXT NOT NULL,
            password TEXT NOT NULL,
            name TEXT,
            active INTEGER NOT NULL DEFAULT 1
        )"
    )
    .execute(pool)
    .await?;

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS image_update_cache (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            stack_name TEXT NOT NULL,
            service_name TEXT NOT NULL,
            image_reference TEXT,
            local_digest TEXT,
            remote_digest TEXT,
            has_update INTEGER DEFAULT 0,
            last_checked INTEGER,
            UNIQUE(stack_name, service_name)
        )"
    )
    .execute(pool)
    .await?;

    // Create knex migration tracking tables so the Node.js backend
    // recognizes that migrations have been applied
    sqlx::query(
        "CREATE TABLE IF NOT EXISTS knex_migrations (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT,
            batch INTEGER,
            migration_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )"
    )
    .execute(pool)
    .await?;

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS knex_migrations_lock (
            index_ INTEGER PRIMARY KEY AUTOINCREMENT,
            is_locked INTEGER
        )"
    )
    .execute(pool)
    .await?;

    info!("Database migrations complete");
    Ok(())
}
