-- 08. Сколько в команде игроков и сколько среди них стримеров.
-- Считаются только незакрытые периоды; команда без состава остаётся
-- в ответе с нулями, стримеры без player_id не учитываются.
--
-- Колонки: name, tag, players_now, streamers_now
-- Порядок: players_now по убыванию, при равенстве name по возрастанию
-- Строк:   24

select name, tag, count(tm.player_id) as players_now, count(s.streamer_id) as streamers_now
from teams t
         left join team_members tm on t.team_id = tm.team_id and tm.left_at is null
         left join streamers s on tm.player_id = s.player_id
group by t.name, t.tag
order by players_now desc, t.name asc