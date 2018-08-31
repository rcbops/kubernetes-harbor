package rackspace

type AuthRequest struct {
	Spec struct {
		Token string `json:"token"`
	} `json:"spec"`
}

type AuthResponse struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Status     struct {
		Authenticated bool   `json:"authenticated"`
		Error         string `json:"error,omitempty"`
		User          struct {
			Username string              `json:"username"`
			UID      string              `json:"uid,omitempty"`
			Groups   []string            `json:"groups,omitempty"`
			Extra    map[string][]string `json:"extra,omitempty"`
		} `json:"user,omitempty"`
	} `json:"status"`
}
