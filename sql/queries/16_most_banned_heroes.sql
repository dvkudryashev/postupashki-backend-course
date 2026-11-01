-- 16. Десятка самых банимых героев.
-- bans — строки match_bans, picks — строки match_players, bans_per_pick =
-- round(bans::numeric / picks, 2); первые десять берутся после сортировки.
--
-- Колонки: hero, bans, picks, bans_per_pick
-- Порядок: bans по убыванию, при равенстве hero по возрастанию
-- Строк:   10

select h.name                                as hero,
       mb.bans                               as bans,
       mp.picks                              as picks,
       round(mb.bans::numeric / mp.picks, 2) as bans_per_pick
from heroes h
         join (select hero_id, count(*) as bans from match_bans group by hero_id) as mb on h.hero_id = mb.hero_id
         join (select hero_id, count(*) as picks from match_players group by hero_id) as mp on h.hero_id = mp.hero_id
order by bans desc, hero asc
limit 10;