-- 15. Сколько донатов собрал каждый стример.
-- В ответе все стримеры: без донатов donations = 0 и total_amount = 0.00;
-- team — команда с left_at IS NULL, иначе пусто.
--
-- Колонки: streamer, platform, team, donations, total_amount
-- Порядок: total_amount по убыванию, при равенстве streamer по возрастанию
-- Строк:   40

select ss.nickname                    as streamer,
       p.name                         as platform,
       t.name                            team,
       count(sd.donation_id)          as donations,
       coalesce(sum(sd.amount), 0.00) as total_amount
from streamers ss
         left join streams s on ss.streamer_id = s.streamer_id
         join platforms p on ss.platform_id = p.platform_id
         left join team_members tm on ss.player_id = tm.player_id and tm.left_at is null
         left join teams t on tm.team_id = t.team_id
         left join stream_donations sd on s.stream_id = sd.stream_id
group by ss.nickname, p.name, t.name
order by total_amount desc, streamer asc;
