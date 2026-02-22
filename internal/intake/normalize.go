package intake

// Normalize validates and normalizes an intake request.
func Normalize(req *Request) error {
	if req == nil {
		return nil
	}
	if req.ProjectID == "" {
		req.ProjectID = "default"
	}
	if req.Priority <= 0 {
		req.Priority = 1
	}
	if req.Source == "" {
		req.Source = "api"
	}
	return nil
}
