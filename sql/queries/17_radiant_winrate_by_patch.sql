-- 17. Доля побед светлой стороны по патчам.
-- Патч без единого матча не выводится; radiant_winrate — процент,
-- round(100.0 * radiant_wins / matches, 2).
--
-- Колонки: version, matches, radiant_wins, radiant_winrate
-- Порядок: radiant_winrate (округлённое) по убыванию, при равенстве version по возрастанию
-- Строк:   12

select p.version,
       m.matches,
       coalesce(wins.radiant_wins, 0)                               as radiant_wins,
       round(100.0 * coalesce(wins.radiant_wins, 0) / m.matches, 2) as radiant_winrate
from patches p
         join (select patch_id, count(*) as matches from matches group by patch_id) as m on p.patch_id = m.patch_id
         left join (select patch_id, count(*) as radiant_wins
                    from matches
                    where winner = 'radiant'
                    group by patch_id) as wins
                   on p.patch_id = wins.patch_id
order by radiant_winrate desc, p.version asc
