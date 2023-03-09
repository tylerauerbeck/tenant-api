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
