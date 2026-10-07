-- Rest after a Pomodoro (GDD §3.2). The server keeps it so every device agrees and it survives a reload.
-- NULL = no rest running (it simply ends when rest_until passes).
ALTER TABLE players ADD COLUMN rest_started_at TEXT;
ALTER TABLE players ADD COLUMN rest_until TEXT;
