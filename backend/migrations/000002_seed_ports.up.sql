-- Insert available port range into port_registry for allocation
-- This is a one-time seed; ports are allocated on demand via FOR UPDATE SKIP LOCKED
-- Range: 3001–9999 (reserved: 80, 443, 8080, 5432, 22, 3000, 2019)

-- Note: This is a one-time seed script. After this, ports are allocated
-- dynamically as projects are created. This ensures the registry knows
-- about available ports without pre-allocating them.

-- We could pre-insert all ports, but it's better to insert on-demand
-- to avoid massive table bloat and unused rows. The application layer
-- handles port allocation with proper locking.
