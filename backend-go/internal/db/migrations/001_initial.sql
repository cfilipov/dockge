CREATE TABLE IF NOT EXISTS setting (
    id     INTEGER PRIMARY KEY,
    key    VARCHAR(200) UNIQUE NOT NULL,
    value  TEXT,
    type   VARCHAR(20)
);

CREATE TABLE IF NOT EXISTS user (
    id               INTEGER PRIMARY KEY,
    username         VARCHAR(255) UNIQUE NOT NULL,
    password         VARCHAR(255),
    active           BOOLEAN DEFAULT 1,
    timezone         VARCHAR(150),
    twofa_secret     VARCHAR(64),
    twofa_status     BOOLEAN DEFAULT 0,
    twofa_last_token VARCHAR(6)
);

CREATE TABLE IF NOT EXISTS agent (
    id       INTEGER PRIMARY KEY,
    url      VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    name     VARCHAR(255),
    active   BOOLEAN DEFAULT 1
);

CREATE TABLE IF NOT EXISTS image_update_cache (
    id               INTEGER PRIMARY KEY,
    stack_name       VARCHAR(255) NOT NULL,
    service_name     VARCHAR(255) NOT NULL,
    image_reference  VARCHAR(500),
    local_digest     VARCHAR(500),
    remote_digest    VARCHAR(500),
    has_update       BOOLEAN DEFAULT 0,
    last_checked     INTEGER,
    UNIQUE(stack_name, service_name)
);
