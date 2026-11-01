-- 25. Место стримера-профессионала в рейтинге своей страны.
-- Только стримеры с player_id и действующей записью в team_members; место по mmr
-- по убыванию и country_players считаются по всей players; player — ник из players.
--
-- Колонки: player, country, mmr, country_rank, country_players, team_tag, platform
-- Порядок: country по возрастанию, внутри страны country_rank по возрастанию
-- Строк:   16

select p.nickname player,
       p.country,
       p.mmr,
       players_stats.country_rank,
       countries.country_players,
       t.tag   as team_tag,
       pl.name as platform
from players p
         join streamers s on p.player_id = s.player_id
         join (select player_id,
                      mmr,
                      rank() over (partition by country order by mmr desc) as country_rank
               from players) as players_stats
              on players_stats.player_id = p.player_id
         join (select country, count(*) as country_players from players group by country) as countries
              on p.country = countries.country
         join team_members tm on p.player_id = tm.player_id and left_at is null
         left join teams t on tm.team_id = t.team_id

         left join platforms pl on s.platform_id = pl.platform_id
order by country asc, country_rank asc