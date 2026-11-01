-- 26. Нарастающая сумма донатов внутри каждого стрима Вжлинка за 2021 год.
-- Стримы Вжлинка, начавшиеся в 2021 году по UTC, строка на каждый донат; running_total —
-- нарастающая сумма донатов своего стрима по sent_at_sec.
--
-- Колонки: stream_id, title, sent_at_sec, donor, amount, running_total
-- Порядок: stream_id по возрастанию, внутри стрима sent_at_sec по возрастанию
-- Строк:   192

select st.stream_id,
       st.title,
       sd.sent_at_sec,
       sd.nickname as donor,
       sd.amount,
       sum(sd.amount) over (
           partition by st.stream_id
           order by sd.sent_at_sec, sd.donation_id
           rows between unbounded preceding and current row
           )       as running_total
from streamers as s
         join streams as st on s.streamer_id = st.streamer_id
         join stream_donations as sd on st.stream_id = sd.stream_id
where s.nickname = 'Вжлинк'
  and st.started_at >= timestamptz '2021-01-01 00:00:00+00'
  and st.started_at < timestamptz '2022-01-01 00:00:00+00'
order by st.stream_id, sd.sent_at_sec, sd.donation_id;