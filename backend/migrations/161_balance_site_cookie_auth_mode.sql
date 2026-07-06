ALTER TABLE balance_sites
    DROP CONSTRAINT IF EXISTS balance_sites_auth_mode_check;

ALTER TABLE balance_sites
    ADD CONSTRAINT balance_sites_auth_mode_check CHECK (auth_mode IN ('password', 'access_token', 'cookie'));

UPDATE balance_sites
SET auth_mode = 'cookie'
WHERE platform = 'newapi'
  AND auth_mode = 'access_token';
