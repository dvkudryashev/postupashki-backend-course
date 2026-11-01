-- 22. Стримеры с подписчиками выше среднего по своей площадке.
-- Строго больше среднего по всем стримерам площадки, сам стример в среднее входит;
-- platform_avg до 2 знаков; team — тег действующей команды, иначе NULL.
--
-- Колонки: streamer, platform, followers, platform_avg, team
-- Порядок: followers по убыванию, при равенстве streamer по возрастанию
-- Строк:   16

with platforms_avg as (select platform_id, avg(followers) as platform_avg from streamers group by platform_id)
select s.nickname                as streamer,
       p.name                    as platform,
       followers,
       round(pa.platform_avg, 2) as platform_avg,
       t.tag                     as team
from streamers s
         join platforms_avg pa on s.platform_id = pa.platform_id
         join platforms p on s.platform_id = p.platform_id
         left join team_members tm on s.player_id = tm.player_id and left_at is null
         left join teams t on tm.team_id = t.team_id
where followers > pa.platform_avg
order by followers desc, streamer asc
