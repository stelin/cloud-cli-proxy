-- Host SSH identity: each host gets its own short_id + entry_password
-- This moves SSH authentication from user-centric to host-centric.
-- SSH login: ssh <host_short_id>@proxy -p 2222 with host's entry_password

ALTER TABLE hosts ADD COLUMN IF NOT EXISTS short_id TEXT UNIQUE;
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS entry_password TEXT NOT NULL DEFAULT '';

-- Migrate existing data: copy entry_password from user to their primary host
UPDATE hosts h
SET entry_password = u.entry_password
FROM users u
WHERE h.user_id = u.id
  AND u.entry_password != ''
  AND h.entry_password = '';

-- Generate short_id for existing hosts that don't have one
-- Uses a simple random approach; uniqueness enforced by UNIQUE constraint
DO $$
DECLARE
  r RECORD;
  new_id TEXT;
  chars TEXT := 'abcdefghijklmnopqrstuvwxyz0123456789';
  i INT;
BEGIN
  FOR r IN SELECT id FROM hosts WHERE short_id IS NULL LOOP
    LOOP
      new_id := '';
      FOR i IN 1..6 LOOP
        new_id := new_id || substr(chars, floor(random() * length(chars) + 1)::int, 1);
      END LOOP;
      BEGIN
        UPDATE hosts SET short_id = new_id WHERE id = r.id;
        EXIT;
      EXCEPTION WHEN unique_violation THEN
        -- retry with different random id
      END;
    END LOOP;
  END LOOP;
END $$;

-- Make short_id NOT NULL after backfill
ALTER TABLE hosts ALTER COLUMN short_id SET NOT NULL;
