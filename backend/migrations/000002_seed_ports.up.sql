-- Seed port pool: 3001–9999 (6999 available ports).
-- Reserved and therefore excluded: 22, 80, 443, 2019, 3000, 5432, 8080.
-- Ports in that reserved list will simply never appear in this table.
INSERT INTO port_registry (port, status)
SELECT gs, 'available'
FROM generate_series(3001, 9999) AS gs
WHERE gs NOT IN (3000, 5432, 8080);
