-- 03. Страны, из которых есть игроки с рейтингом не ниже 10000.
-- mmr >= 10000; каждая страна в ответе ровно один раз.
--
-- Колонки: country
-- Порядок: country по возрастанию
-- Строк:   15

select country
from players
where mmr >= '10000'
group by country
order by country asc;
