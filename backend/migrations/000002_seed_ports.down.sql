DELETE FROM port_registry WHERE status = 'available' AND project_id IS NULL;
