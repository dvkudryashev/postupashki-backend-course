-- 20. Герои, которых на патче банили чаще, чем брали.
-- Патч — matches.patch_id; в ответе пары с bans строго больше picks, diff = bans - picks.
--
-- Колонки: version, hero, bans, picks, diff
-- Порядок: diff по убыванию, затем hero по возрастанию, затем version по возрастанию
-- Строк:   99

with bans_by_patch as
         (select m.patch_id,
                 mb.hero_id,
                 count(*) as bans
          from match_bans mb
                   join matches m on mb.match_id = m.match_id
          group by m.patch_id, mb.hero_id),
     picks_by_patch as
         (select m.patch_id,
                 mp.hero_id,
                 count(*) as picks
          from match_players mp
                   join matches m on mp.match_id = m.match_id
          group by m.patch_id, mp.hero_id)

select p.version                        as version,
       h.name                           as hero,
       bbp.bans                         as bans,
       coalesce(pbp.picks, 0)            as picks,
       bbp.bans - coalesce(pbp.picks, 0) as diff
from bans_by_patch bbp
         left join picks_by_patch pbp on bbp.hero_id = pbp.hero_id and bbp.patch_id = pbp.patch_id
         join patches p on bbp.patch_id = p.patch_id
         join heroes h on bbp.hero_id = h.hero_id
where bbp.bans > coalesce(pbp.picks, 0)
order by diff desc, hero asc, version asc
