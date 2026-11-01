-- 09. Кто из стримеров за какую команду выступает сейчас.
-- Только действующие составы; стримеры без player_id не выводятся.
--
-- Колонки: streamer, platform, player, team, tag, followers
-- Порядок: followers по убыванию, при равенстве streamer по возрастанию
-- Строк:   16

select s.nickname as streamer, p.name as platform, pl.nickname as player, t.name as team, tag, followers
from streamers s
         join platforms p on s.platform_id = p.platform_id and s.player_id is not null
         join team_members tm on s.player_id = tm.player_id and tm.left_at is null
         left join players pl on s.player_id = pl.player_id
         join teams t on tm.team_id = t.team_id
order by followers desc, streamer asc
