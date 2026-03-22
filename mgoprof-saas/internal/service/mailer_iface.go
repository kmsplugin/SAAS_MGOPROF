package service

// RegistrationMailer is the narrow interface for emails sent during the
// registration flow. Using an interface instead of *mailer.Mailer lets tests
// inject a controllable fake without touching SMTP.
type RegistrationMailer interface {
	// SendRegistration delivers the 6-digit OTP to the registrant.
	SendRegistration(to, firstName, otp, eventTitle string) error
	// SendWelcome delivers credentials and cabinet link after OTP success.
	SendWelcome(to, firstName, password string, userID, regID int, cabinetURL, ticketURL string) error
}

// AuthMailer is the narrow interface for emails sent from AuthService.
type AuthMailer interface {
	// SendNewPassword delivers a freshly generated password to the user.
	SendNewPassword(to, firstName, password, loginURL string) error
	// SendDataExport delivers a GDPR/152-ФЗ data export to the user.
	SendDataExport(to, firstName, dataJSON string) error
}
