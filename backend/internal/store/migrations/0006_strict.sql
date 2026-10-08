-- Strict mode (GDD §3.3): a personal challenge chosen when a Pomodoro starts. It never blocks or costs anything; a completed
-- strict Pomodoro that kept to the limits counts as "clean" in the Harvest Book.
ALTER TABLE pomodoros ADD COLUMN strict INTEGER NOT NULL DEFAULT 0;
