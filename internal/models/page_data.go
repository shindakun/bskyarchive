package models

// LoginPageData represents the data passed to the login template for rendering.
// It contains all information needed to display the login page with proper error
// handling and form repopulation.
type LoginPageData struct {
	// Title is the page title displayed in the browser tab and page header
	Title string

	// Error contains the error message to display when login fails or validation errors occur.
	// Empty string means no error.
	Error string

	// Message contains an informational message (rarely used for login page, but included for consistency with other pages).
	// Empty string means no message.
	Message string

	// Handle is the Bluesky handle value to pre-populate in the form.
	// Used for repopulating the form after validation errors so users don't have to re-type.
	Handle string
}
