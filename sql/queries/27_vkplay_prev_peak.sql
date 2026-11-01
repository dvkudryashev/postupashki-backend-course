-- 27. Пик онлайна против пика предыдущего эфира того же стримера.
-- Только площадка 'VK Play Live'; prev_peak — пик предыдущего по started_at эфира того
-- же стримера, delta — их разность; первый эфир каждого стримера не выводится.
--
-- Колонки: streamer, followers, started_at, title, peak_viewers, prev_peak, delta
-- Порядок: followers по убыванию, внутри стримера started_at по возрастанию
-- Строк:   592

with stream_stats as
         (select s.nickname as streamer,
                 s.followers,
                 st.started_at,
                 st.title,
                 st.peak_viewers,
                 lag(st.peak_viewers) over (
                     partition by s.streamer_id
                     order by st.started_at
                     )      as prev_peak
          from streamers as s
                   join platforms as p on s.platform_id = p.platform_id
                   join streams as st on s.streamer_id = st.streamer_id
          where p.name = 'VK Play Live')

select streamer,
       followers,
       started_at,
       title,
       peak_viewers,
       prev_peak,
       peak_viewers - prev_peak as delta
from stream_stats
where prev_peak is not null
order by followers desc, started_at asc;
