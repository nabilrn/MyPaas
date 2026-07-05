INSERT INTO users (email, role)
VALUES ('2211522018_nabil@student.unand.ac.id', 'owner')
ON CONFLICT (email) DO NOTHING;
