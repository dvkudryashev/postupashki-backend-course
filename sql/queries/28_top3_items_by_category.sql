-- 28. Три самых дорогих предмета в каждой категории лавки.
-- Категория: 'recipe' при is_recipe, иначе 'assembled' при строках в item_recipe.item_id,
-- иначе 'basic'; place 1-3 по cost убыв., при равной цене по name; purchases — строк match_items, иначе 0.
--
-- Колонки: category, place, name, cost, purchases
-- Порядок: category по возрастанию, затем place по возрастанию
-- Строк:   9

with categorized_items as (select i.item_id,
                                  i.name,
                                  i.cost,
                                  case
                                      when i.is_recipe then 'recipe'
                                      when exists (select 1
                                                   from item_recipe as ir
                                                   where ir.item_id = i.item_id) then 'assembled'
                                      else 'basic'
                                      end as category
                           from items as i),
     ranked_items as (select ci.*,
                             row_number() over (
                                 partition by ci.category
                                 order by ci.cost desc, ci.name asc
                                 ) as place
                      from categorized_items as ci),
     purchase_counts as (select item_id,
                                count(*) as purchases
                         from match_items
                         group by item_id)
select ri.category,
       ri.place,
       ri.name,
       ri.cost,
       coalesce(pc.purchases, 0) as purchases
from ranked_items as ri
         left join purchase_counts as pc on ri.item_id = pc.item_id
where ri.place between 1 and 3
order by ri.category, ri.place;