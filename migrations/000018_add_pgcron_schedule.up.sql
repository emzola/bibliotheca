CREATE EXTENSION IF NOT EXISTS pg_cron;

-- set a cron schedule that resets user's daily download count to 0
cron.schedule(
    'dlcount-reset-everyday', -- name of the cron job
    '0 0 * * *', -- every day at 00:00
    $$
    UPDATE users 
    SET download_count = 0;
    $$
);

-- set a cron schedule that marks requests older than 6 months as expired
cron.schedule(
    'expire-requests-every-5mins', -- name of the cron job
    '*/5 * * * *', -- every 5 minutes
    $$
    UPDATE requests 
    SET status = 'expired'
    WHERE expiry > CURRENT_TIMESTAMP(0);
    $$
);