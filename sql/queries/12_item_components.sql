-- 12. Сборные предметы внутри сборных предметов.
-- Пары «предмет — непосредственный компонент», где компонент сам встречается
-- в item_recipe.item_id; каждая пара один раз, component_cost — цена штуки.
--
-- Колонки: item_name, component_name, quantity, component_cost
-- Порядок: item_name по возрастанию, внутри предмета component_name по возрастанию
-- Строк:   36

select item.name as item_name, component.name component_name, link.quantity, component.cost component_cost
from items item
         join item_recipe link on item.item_id = link.item_id
         join items component on component.item_id = link.component_id
where exists(select 1
             from item_recipe as component_recipe
             where component_recipe.item_id = component.item_id)

order by item_name asc, component_name asc