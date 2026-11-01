-- 01. Герои ближнего боя с основным атрибутом «сила».
-- primary_attr = 'str' и attack_type = 'melee' одновременно.
--
-- Колонки: hero_id, name, legs
-- Порядок: name по возрастанию
-- Строк:   11

select hero_id, name, legs from heroes
where primary_attr = 'str' and attack_type = 'melee'
order by name asc;
