-- 10. Все роли героев, которые умеют быть керри.
-- Только герои с ролью 'carry'; у каждого выводятся все его роли, по строке на роль.
--
-- Колонки: name, primary_attr, attack_type, role
-- Порядок: name по возрастанию, внутри героя role по возрастанию
-- Строк:   47

select name, primary_attr, attack_type, hr2.role
from heroes h
         join hero_roles hr on h.hero_id = hr.hero_id
         join hero_roles hr2 on hr.hero_id = hr2.hero_id
where hr.role in ('carry')
order by name asc, role asc