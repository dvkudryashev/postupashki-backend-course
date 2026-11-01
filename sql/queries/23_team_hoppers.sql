-- 23. Игроки, поигравшие больше чем за одну команду.
-- Считаются различные team_id по всем периодам, строго больше одного; current_team —
-- название команды с пустым left_at, current_team и streamer могут быть NULL.
--
-- Колонки: player, country, teams_count, current_team, streamer
-- Порядок: teams_count по убыванию, при равенстве player по возрастанию
-- Строк:   25

with player_teams as (select player_id, count(distinct team_id) as teams_count
                      from team_members
                      group by player_id
                      having count(distinct team_id) > 1)
select p.nickname as player,
       p.country  as country,
       pt.teams_count,
       t.name     as current_team,
       s.nickname as streamer
from player_teams pt
         join players p on pt.player_id = p.player_id
         left join team_members tm on pt.player_id = tm.player_id and left_at is null
         left join streamers s on pt.player_id = s.player_id
         left join teams t on tm.team_id = t.team_id
order by teams_count desc, player asc