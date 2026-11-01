-- 07. Действующие составы: кто сейчас за кого играет.
-- Только незакрытые периоды (left_at IS NULL); игроки без команды не выводятся.
--
-- Колонки: nickname, country, mmr, team_name, joined_at
-- Порядок: joined_at по возрастанию, при равенстве nickname по возрастанию
-- Строк:   86

select p.nickname, p.country, p.mmr, t.name as team_name, tm.joined_at
from teams t
join team_members tm on t.team_id = tm.team_id
left join players p on tm.player_id = p.player_id
where tm.left_at is null
order by joined_at asc, nickname asc;