-- 05. Герои с необычным числом ног.
-- legs <> 2; безногие герои тоже подходят.
--
-- Колонки: name, primary_attr, attack_type, legs
-- Порядок: legs по убыванию, при равенстве name по возрастанию
-- Строк:   17

select name, primary_attr, attack_type, legs
from heroes
where legs <> 2
order by legs desc, name asc;