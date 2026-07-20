ALTER TABLE agent_controllers ADD COLUMN description text NOT NULL DEFAULT '';
ALTER TABLE agent_controllers ADD COLUMN tls_hostname text NOT NULL DEFAULT '';
ALTER TABLE agent_controllers ADD COLUMN tags jsonb NOT NULL DEFAULT '[]';
ALTER TABLE agent_controllers ADD COLUMN is_active boolean NOT NULL DEFAULT true;
ALTER TABLE agent_controllers ADD COLUMN updated_at timestamp with time zone NOT NULL DEFAULT now();
