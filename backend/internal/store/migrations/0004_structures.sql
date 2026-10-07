-- Structures (GDD §4.7): one hive per plot cell, a single Pet Dog per player.
-- These indexes make the rules hold even if two requests race.
CREATE UNIQUE INDEX one_hive_per_cell ON structures(player_id, x, y) WHERE kind = 'hive';
CREATE UNIQUE INDEX one_dog_per_player ON structures(player_id) WHERE kind = 'dog';
