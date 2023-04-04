package api

type createTenantRequest struct {
	Name string `json:"name"`
}

func (c *createTenantRequest) validate() error {
	if c.Name == "" {
		return ErrTenantNameMissing
	}

	return nil
}

type updateTenantRequest struct {
	Name *string `json:"name"`
}

func (c *updateTenantRequest) validate() error {
	if c.Name != nil && *c.Name == "" {
		return ErrTenantNameMissing
	}

	return nil
}
