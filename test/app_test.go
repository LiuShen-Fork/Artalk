package test

import "testing"

func TestNewTestApp(t *testing.T) {
	app, err := NewTestApp()
	if app == nil || err != nil {
		t.Fatal(err)
	}

	if err := app.Cleanup(); err != nil {
		t.Fatal(err)
	}
}

// TestAdminLinkFromConfIsNormalized guards the link an admin writes in the
// config. Every handler that saves a comment or a user refuses a link that
// ValidateURL rejects, so a bare host stored here would come back as a 400 the
// next time the record is saved from the dashboard.
func TestAdminLinkFromConfIsNormalized(t *testing.T) {
	app, err := NewTestApp()
	if app == nil || err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	user := app.Dao().FindUser("admin_in_conf", "admin_in_conf@example.org")
	if user.IsEmpty() {
		t.Fatal("admin from config was not created")
	}

	const want = "https://admin_in_conf.example.org"
	if user.Link != want {
		t.Errorf("admin link = %q, want %q", user.Link, want)
	}
}
