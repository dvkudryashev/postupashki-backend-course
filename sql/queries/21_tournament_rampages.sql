-- 21. Турнирные выступления с KDA выше шестикратного среднего.
-- KDA = (kills + assists)::numeric / GREATEST(deaths, 1); порог — строго больше шести
-- средних KDA по всей match_players; только матчи турниров; kda округлить до 2 знаков.
--
-- Колонки: match_id, tournament, player, hero, kda
-- Порядок: kda по убыванию, при равенстве match_id, затем slot
-- Строк:   60

select m.match_id,
       t.name                                                     as tournament,
       p.nickname                                                 as player,
       h.name                                                     as hero,
       round((kills + assists)::numeric / greatest(deaths, 1), 2) as kda
from matches m
         join tournaments t on m.tournament_id = t.tournament_id
         join match_players mp on m.match_id = mp.match_id
         join players p on mp.player_id = p.player_id
         join heroes h on mp.hero_id = h.hero_id
where (kills + assists)::numeric / greatest(deaths, 1) >
      6 * (select avg((kills + assists)::numeric / greatest(deaths, 1)) from match_players)
order by kda desc, match_id asc, slot asc