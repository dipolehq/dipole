ALTER TABLE server_workloads ADD COLUMN transport_protocol TEXT NOT NULL DEFAULT 'TCP';
ALTER TABLE server_workloads ADD COLUMN tls_verification TEXT NOT NULL DEFAULT 'full';
ALTER TABLE server_workloads ADD COLUMN requested_port INT NOT NULL DEFAULT 0;
ALTER TABLE server_workloads ADD COLUMN requested_tls BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE server_workloads ADD COLUMN http_headers JSONB NOT NULL DEFAULT '{}';
