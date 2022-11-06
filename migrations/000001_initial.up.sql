-- sqlite --
CREATE TABLE users (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT,
  "email" VARYING CHARACTER (254) NOT NULL UNIQUE,
  "password" VARYING CHARACTER (80) NOT NULL,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE links (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT,
  "url" VARYING CHARACTER NOT NULL UNIQUE,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_links (
  "user_id" INTEGER NOT NULL,
  "link_id" INTEGER NOT NULL,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("user_id", "link_id"),
  FOREIGN KEY ("user_id") REFERENCES users ("id") ON DELETE CASCADE,
  FOREIGN KEY ("link_id") REFERENCES links ("id") ON DELETE CASCADE
);

CREATE TABLE logins (
  "user_id" INTEGER NOT NULL,
  "token" VARYING CHARACTER (38) NOT NULL UNIQUE,
  "active_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("user_id", "token")
);
-- end sqlite --

-- begin postgresql --
--CREATE TABLE users (
--  "id" SERIAL PRIMARY KEY,
--  "email" character varying (254) NOT NULL UNIQUE,
--  "password" character varying (80),
--  "created_at" timestamp (0) NOT NULL DEFAULT (now() at time zone 'utc')
--);
--
--CREATE TABLE links (
--  "id" SERIAL PRIMARY KEY,
--  "url" character varying NOT NULL UNIQUE,
--  "created_at" timestamp (0) NOT NULL DEFAULT (now() at time zone 'utc')
--);
--
--CREATE TABLE user_links (
--  "user_id" integer NOT NULL REFERENCES users ON DELETE CASCADE,
--  "link_id" integer NOT NULL REFERENCES links ON DELETE CASCADE,
--  "created_at" timestamp (0) NOT NULL DEFAULT (now() at time zone 'utc'),
--  PRIMARY KEY ("user_id", "link_id")
--);
--
--CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
--CREATE TABLE logins (
--    "user_id" integer NOT NULL REFERENCES users ON DELETE CASCADE,
--    "token" UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
--    "active_at" timestamp (0) NOT NULL DEFAULT (now() at time zone 'utc'),
--    "created_at" timestamp (0) NOT NULL DEFAULT (now() at time zone 'utc'),
--    PRIMARY KEY ("user_id", "token")
--);
-- end postgresql --
