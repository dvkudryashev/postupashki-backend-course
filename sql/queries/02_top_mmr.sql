-- 02. Пятнадцать самых высокорейтинговых игроков выше 9000.
-- mmr строго больше 9000; пятнадцать строк отсекаются после сортировки.
--
-- Колонки: nickname, country, mmr
-- Порядок: mmr по убыванию, при равенстве nickname по возрастанию
-- Строк:   15

select nickname, country, mmr
from players
where mmr > '9000'
order by mmr desc, nickname asc
limit 15