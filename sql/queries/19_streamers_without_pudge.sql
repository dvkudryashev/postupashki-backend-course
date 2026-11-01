-- 19. Стримеры, которые ни разу не брали Pudge.
-- Только стримеры с непустым player_id и без единой игры за 'Pudge'; team — тег
-- команды с left_at IS NULL, иначе пусто; games — все строки match_players игрока.
--
-- Колонки: streamer, platform, team, followers, games
-- Порядок: followers по убыванию, при равенстве streamer по возрастанию
-- Строк:   8

select s.nickname         as streamer,
       pl.name            as platform,
       t.tag              as team,
       followers,
       count(mp.match_id) as games
from streamers s
         left join match_players mp on s.player_id = mp.player_id
         join platforms pl on s.platform_id = pl.platform_id
         left join team_members on s.player_id = team_members.player_id and left_at is null
         left join teams t on team_members.team_id = t.team_id
where s.player_id is not null
  and not exists (select 1
                  from match_players as pudge_games
                           join heroes h on pudge_games.hero_id = h.hero_id
                  where pudge_games.player_id = s.player_id
                    and h.name = 'Pudge')
group by s.nickname, pl.name, t.tag, followers
order by s.followers desc, streamer asc
