CREATE TABLE role_resource_set_assignments (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource_set_id UUID NOT NULL REFERENCES resource_sets(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, resource_set_id)
);
