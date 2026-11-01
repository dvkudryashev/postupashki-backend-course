-- 13. Сколько матчей сыграно в каждом году.
-- Год из started_at; tournament_matches — матчи с непустым tournament_id, ноль допустим.
--
-- Колонки: year, matches, tournament_matches
-- Порядок: year по возрастанию
-- Строк:   10

select extract(year from m.started_at) as                           year,
       count(m.match_id)               as                           matches,
       count(m.match_id) filter (where m.tournament_id is not null) tournament_matches
from matches m
         left join tournaments t on m.tournament_id = t.tournament_id
group by year
order by year asc;