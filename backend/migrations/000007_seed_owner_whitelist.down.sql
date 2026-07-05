DELETE FROM users
WHERE email = '2211522018_nabil@student.unand.ac.id'
  AND github_id IS NULL
  AND github_username IS NULL
  AND avatar_url IS NULL
  AND last_login_at IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM projects
      WHERE projects.user_id = users.id
  );
