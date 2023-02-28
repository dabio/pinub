CREATE TABLE IF NOT EXISTS users (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT,
  "email" VARYING CHARACTER (254) NOT NULL UNIQUE,
  "password" VARYING CHARACTER (80) NOT NULL,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS links (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT,
  "url" VARYING CHARACTER NOT NULL UNIQUE,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_links (
  "user_id" INTEGER NOT NULL,
  "link_id" INTEGER NOT NULL,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("user_id", "link_id"),
  FOREIGN KEY ("user_id") REFERENCES users ("id") ON DELETE CASCADE,
  FOREIGN KEY ("link_id") REFERENCES links ("id") ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS logins (
  "user_id" INTEGER NOT NULL,
  "token" VARYING CHARACTER (38) NOT NULL UNIQUE,
  "active_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("user_id", "token")
);