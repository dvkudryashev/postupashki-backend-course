-- 30. Стоимость сборки из базовых составляющих против собственной цены предмета.
-- Берутся предметы со строками в item_recipe, сборка раскрывается до базовых (тех, у кого
-- строк в item_recipe нет, свитки тоже), количества по пути перемножаются; diff = parts_cost - cost.
--
-- Колонки: name, cost, parts_cost, base_parts, diff
-- Порядок: diff по убыванию, при равенстве name по возрастанию
-- Строк:   77

with recursive
    recipe_tree as (select root.item_id      as root_id,
                           component.item_id as component_id,
                           ir.quantity       as qty_total
                    from items as root
                             join item_recipe as ir on root.item_id = ir.item_id
                             join items as component on ir.component_id = component.item_id

                    union all

                    select tree.root_id,
                           component.item_id,
                           tree.qty_total * ir.quantity
                    from recipe_tree as tree
                             join item_recipe as ir on tree.component_id = ir.item_id
                             join items as component on ir.component_id = component.item_id),
    leaf_totals as (select tree.root_id,
                           sum(component.cost * tree.qty_total) as parts_cost,
                           sum(tree.qty_total)                  as base_parts
                    from recipe_tree as tree
                             join items as component on tree.component_id = component.item_id
                    where not exists (select 1
                                      from item_recipe as ir
                                      where ir.item_id = tree.component_id)
                    group by tree.root_id)
select root.name,
       root.cost,
       totals.parts_cost,
       totals.base_parts,
       totals.parts_cost - root.cost as diff
from leaf_totals as totals
         join items as root on totals.root_id = root.item_id
order by diff desc, root.name asc;