-- 24. Турниры, чей призовой фонд больше суммы призовых всех предыдущих.
-- Предыдущие — со строго меньшим started_at, сравнение строгое; у первого турнира
-- prev_total и prev_count нули, не NULL; matches_played — число матчей турнира.
--
-- Колонки: tournament, started_at, prize_pool, prev_total, prev_count, matches_played
-- Порядок: started_at по возрастанию
-- Строк:   2

with tournament_stats as
         (select current.tournament_id,
                 current.name,
                 current.started_at,
                 current.prize_pool,

                 coalesce((select sum(previous.prize_pool)
                           from tournaments as previous
                           where previous.started_at < current.started_at), 0) as prev_total,

                 (select count(*)
                  from tournaments as previous
                  where previous.started_at < current.started_at)              as prev_count,

                 (select count(*)
                  from matches as m
                  where m.tournament_id = current.tournament_id)               as matches_played

          from tournaments as current)

select name as tournament,
       started_at,
       prize_pool,
       prev_total,
       prev_count,
       matches_played
from tournament_stats
where prize_pool > prev_total
order by started_at asc;