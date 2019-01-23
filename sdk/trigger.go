package sdk

// Prerequisite defines a expected value to one triggering pipeline parameter
type Prerequisite struct {
	Parameter     string `json:"parameter"`
	ExpectedValue string `json:"expected_value"`
}
