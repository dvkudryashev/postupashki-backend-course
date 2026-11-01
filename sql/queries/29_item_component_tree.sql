-- 29. Полное дерево сборки Guardian Greaves, Bloodthorn, Harpoon и Hurricane Pike.
-- Строка на каждый путь вниз, сам корень не выводится; lvl с 1, qty — quantity последнего
-- шага, qty_total — произведение quantity по пути, path — имена от корня через ' > '.
--
-- Колонки: root_item, lvl, component, qty, qty_total, path
-- Порядок: root_item по возрастанию, затем path по возрастанию
-- Строк:   34

with recursive recipe_tree as (select root.name                            as root_item,
                                      component.item_id                    as component_id,
                                      component.name                       as component,
                                      1                                    as lvl,
                                      ir.quantity                          as qty,
                                      ir.quantity                          as qty_total,
                                      root.name || ' > ' || component.name as path
                               from items as root
                                        join item_recipe as ir on root.item_id = ir.item_id
                                        join items as component on ir.component_id = component.item_id
                               where root.name in (
                                                   'Guardian Greaves',
                                                   'Bloodthorn',
                                                   'Harpoon',
                                                   'Hurricane Pike'
                                   )

                               union all

                               select tree.root_item,
                                      component.item_id,
                                      component.name,
                                      tree.lvl + 1,
                                      ir.quantity,
                                      tree.qty_total * ir.quantity,
                                      tree.path || ' > ' || component.name
                               from recipe_tree as tree
                                        join item_recipe as ir
                                             on tree.component_id = ir.item_id
                                        join items as component on ir.component_id = component.item_id)

select root_item,
       lvl,
       component,
       qty,
       qty_total,
       path
from recipe_tree
order by root_item, path;