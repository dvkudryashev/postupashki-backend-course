-- 18. Призовые турниров, до которых добрались игроки команды.
-- Турнир в зачёт один раз, если игрок сыграл его матч в составе: joined_at <=
-- started_at::date < left_at (пустой left_at — сейчас); команды без турниров не выводятся.
--
-- Колонки: team, tag, tournaments, prize_total
-- Порядок: prize_total по убыванию, при равенстве team по возрастанию
-- Строк:   22

with tournament_participants as
         (select distinct t.team_id, tr.tournament_id
          from teams t
                   join team_members tm on t.team_id = tm.team_id
                   join match_players mp on tm.player_id = mp.player_id
                   join matches m on mp.match_id = m.match_id
                   join tournaments tr on m.tournament_id = tr.tournament_id
          where tm.joined_at <= m.started_at::date
            and (tm.left_at is null or m.started_at::date < tm.left_at))

select t.name                  as team,
       t.tag                   as tag,
       count(tr.tournament_id) as tournaments,
       sum(tr.prize_pool)      as prize_total
from tournament_participants tp
         join tournaments tr on tp.tournament_id = tr.tournament_id
         join teams t on t.team_id = tp.team_id
group by t.name, t.tag
order by prize_total desc, team asc