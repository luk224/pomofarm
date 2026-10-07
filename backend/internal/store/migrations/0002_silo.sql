-- Silo (GDD §3.4, §4.5, §6.2): 🪙 produced by mature plants wait here until collected.
-- Millionths of a coin (INTEGER): short, frequent settlements must not lose fractions to rounding.
ALTER TABLE players ADD COLUMN silo_micro INTEGER NOT NULL DEFAULT 0;
-- Highest production rate (millionths of a coin per hour) seen since the Silo was last emptied.
-- The Silo holds at most `capacity hours × this rate` (GDD §6.2: "ritmo máximo de la ventana").
ALTER TABLE players ADD COLUMN silo_peak_micro_h INTEGER NOT NULL DEFAULT 0;
