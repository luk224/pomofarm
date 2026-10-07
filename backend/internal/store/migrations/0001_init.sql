-- Importes de 🪙 en milésimas (INTEGER) para no perder fracciones.
CREATE TABLE players (
  id              INTEGER PRIMARY KEY,
  name            TEXT    NOT NULL,
  focus_points    INTEGER NOT NULL DEFAULT 0,
  lifetime_focus  INTEGER NOT NULL DEFAULT 0,
  coins_milli     INTEGER NOT NULL DEFAULT 0,
  silo_level      INTEGER NOT NULL DEFAULT 0,
  season          INTEGER NOT NULL DEFAULT 1,
  biome           TEXT    NOT NULL DEFAULT 'spring',
  created_at      TEXT    NOT NULL,
  last_seen_at    TEXT    NOT NULL,
  version         INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE unlocks (
  player_id INTEGER NOT NULL REFERENCES players(id),
  kind      TEXT    NOT NULL,       -- seed | animal | cosmetic
  key       TEXT    NOT NULL,       -- tomato | sunflower | bees | dog ...
  at        TEXT    NOT NULL,
  PRIMARY KEY (player_id, kind, key)
);

CREATE TABLE plots (
  id            INTEGER PRIMARY KEY,
  player_id     INTEGER NOT NULL REFERENCES players(id),
  x INTEGER NOT NULL, y INTEGER NOT NULL,
  plant_type    TEXT,
  state         TEXT NOT NULL DEFAULT 'empty', -- empty | growing | mature | withered
  planted_at    TEXT,
  grow_s        INTEGER,                       -- duración del Pomodoro
  matured_at    TEXT,
  harvested     INTEGER NOT NULL DEFAULT 0,
  life_s        INTEGER,                       -- vida útil
  wilts_at      TEXT,                          -- matured_at + life_s
  collected_to  TEXT,                          -- hasta cuándo se han contado sus 🪙
  version       INTEGER NOT NULL DEFAULT 0,
  UNIQUE (player_id, x, y)
);

CREATE TABLE tags (
  id INTEGER PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(id),
  name TEXT NOT NULL,
  UNIQUE (player_id, name)
);

CREATE TABLE pomodoros (
  id            INTEGER PRIMARY KEY,
  player_id     INTEGER NOT NULL REFERENCES players(id),
  plot_id       INTEGER REFERENCES plots(id),
  plant_type    TEXT    NOT NULL,
  tag_id        INTEGER REFERENCES tags(id),
  planned_s     INTEGER NOT NULL,
  started_at    TEXT    NOT NULL,
  paused_at     TEXT,                    -- no nulo mientras está pausado
  paused_total_s INTEGER NOT NULL DEFAULT 0,
  ended_at      TEXT,
  reward_focus  INTEGER NOT NULL DEFAULT 0,
  status        TEXT    NOT NULL         -- running | paused | completed | cancelled
);
-- Un solo Pomodoro activo por jugador:
CREATE UNIQUE INDEX one_active_pomodoro
  ON pomodoros(player_id) WHERE status IN ('running','paused');

CREATE TABLE pomodoro_events (           -- auditoría de pausas
  id INTEGER PRIMARY KEY,
  pomodoro_id INTEGER NOT NULL REFERENCES pomodoros(id),
  kind TEXT NOT NULL,                    -- start | pause | resume | complete | cancel
  at TEXT NOT NULL
);

CREATE TABLE structures (
  id INTEGER PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(id),
  kind TEXT NOT NULL,                    -- hive | dog | path | fence | lantern ...
  x INTEGER NOT NULL, y INTEGER NOT NULL
);

CREATE TABLE settings (
  player_id INTEGER NOT NULL REFERENCES players(id),
  key TEXT NOT NULL, value TEXT NOT NULL,
  PRIMARY KEY (player_id, key)
);
