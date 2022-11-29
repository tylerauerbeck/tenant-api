package checker

import "errors"

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) CanAccess() bool {
	return true
}

func (c *Client) Scopes() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ResourcesAvailable(resourceTypeURN string, scope string, actor any) ([]string, error) {
	if resourceTypeURN == "tenant" {
		return []string{"d319a53a-83c1-4a57-831c-0188952005d6", "b3d6ce63-9cda-4ecd-9ca0-4adfd33e1798"}, nil
	}

	return []string{}, errors.New("invalid resource type")
}

func (c *Client) ActorHasScope(actor any, scope string, resType string, resID string) bool {
	return true
}

func (c *Client) ActorHasGlobalScope(actor any, scope string) bool {
	return true
}
