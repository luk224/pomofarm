-- Decoration (GDD §4.7): at most one piece per background cell, one hat per player. Decoration never sits on a plot cell
-- (the service checks that) and never counts for prestige.
CREATE UNIQUE INDEX one_decor_per_cell ON structures(player_id, x, y) WHERE kind IN ('path', 'fence', 'lantern');
CREATE UNIQUE INDEX one_hat_per_player ON structures(player_id) WHERE kind = 'hat';
