package license

// LicenseStatus represents the result of license validation
type LicenseStatus struct {
	Valid   bool            // true if signature OK and not expired
	Payload *LicensePayload // parsed license data
	Error   error           // error encountered, if any
}

// Check runs validation and returns a LicenseStatus
func Check(licensePath, pubKeyPath string) *LicenseStatus {
	payload, err := ValidateLicense(licensePath, pubKeyPath)
	if err != nil {
		return &LicenseStatus{
			Valid:   false,
			Payload: nil,
			Error:   err,
		}
	}

	return &LicenseStatus{
		Valid:   true,
		Payload: payload,
		Error:   nil,
	}
}
