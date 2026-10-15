package mux

import "fmt"

// Tab groups the sessions opened to one server. Each session in a tab
// belongs to a different SSH user (e.g. irteam, irteamsu); only the
// active one is shown and receives input.
type Tab struct {
	Name string // server display name (hostname)
	Host string // ssh target (hostname or IP)
	Dup  int    // 1 for the first tab of a server, 2+ for duplicates

	sessions []*Session // at most one per user, in open order
	active   int
}

// Current returns the session currently shown in the tab.
func (t *Tab) Current() *Session {
	if t.active < 0 || t.active >= len(t.sessions) {
		return nil
	}
	return t.sessions[t.active]
}

// CurrentUser returns the user of the active session, or "" if the tab is empty.
func (t *Tab) CurrentUser() string {
	if s := t.Current(); s != nil {
		return s.User
	}
	return ""
}

// Users returns the users that have a session in this tab, in open order.
func (t *Tab) Users() []string {
	users := make([]string, 0, len(t.sessions))
	for _, s := range t.sessions {
		users = append(users, s.User)
	}
	return users
}

// Sessions returns the tab's sessions.
func (t *Tab) Sessions() []*Session {
	return t.sessions
}

// DisplayName is the server name with a "#N" suffix for duplicate tabs.
func (t *Tab) DisplayName() string {
	if t.Dup > 1 {
		return fmt.Sprintf("%s #%d", t.Name, t.Dup)
	}
	return t.Name
}

// activate switches the tab to the given user's session. Returns false if
// the tab has no session for that user.
func (t *Tab) activate(user string) bool {
	for i, s := range t.sessions {
		if s.User == user {
			t.active = i
			return true
		}
	}
	return false
}

// add appends a session and makes it active.
func (t *Tab) add(s *Session) {
	t.sessions = append(t.sessions, s)
	t.active = len(t.sessions) - 1
}

// remove drops the session from the tab, keeping a sensible active index.
func (t *Tab) remove(s *Session) {
	for i, cur := range t.sessions {
		if cur != s {
			continue
		}
		t.sessions = append(t.sessions[:i], t.sessions[i+1:]...)
		if t.active > i || t.active >= len(t.sessions) {
			t.active--
		}
		if t.active < 0 && len(t.sessions) > 0 {
			t.active = 0
		}
		return
	}
}

// otherUser returns the user Ctrl+A u switches to from the given user.
func otherUser(user string) string {
	if user == "irteamsu" {
		return "irteam"
	}
	return "irteamsu"
}
