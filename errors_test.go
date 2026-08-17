package gsxmail_test

import (
	"errors"
	"testing"
	"testing/fstest"

	"m31labs.dev/gsxmail"
)

// TestErrCompile is polish item 6's own proof for ErrCompile: a *.gsx
// file that does not parse as gosx source at all.
func TestErrCompile(t *testing.T) {
	fsys := fstest.MapFS{
		"broken.gsx": {Data: []byte(`package emails

func Broken(props Props) Node {
    return <email.Shell wordmark="x"
}
`)},
	}
	_, err := gsxmail.Load(fsys, gsxmail.Options{})
	if err == nil {
		t.Fatal("Load succeeded; broken.gsx is not valid gosx source")
	}
	if !errors.Is(err, gsxmail.ErrCompile) {
		t.Errorf("errors.Is(err, ErrCompile) = false; err = %v", err)
	}
}

// TestErrDuplicateTemplate is polish item 6's own proof for
// ErrDuplicateTemplate: two files declaring the same component name.
func TestErrDuplicateTemplate(t *testing.T) {
	src := `package emails

func Dup() Node {
    return <email.Shell wordmark="x" shortCode="X" tagline="T" title="T" lang="en" preheader="p">
        <email.Note text="n" />
    </email.Shell>
}
`
	fsys := fstest.MapFS{
		"a.gsx": {Data: []byte(src)},
		"b.gsx": {Data: []byte(src)},
	}
	_, err := gsxmail.Load(fsys, gsxmail.Options{})
	if err == nil {
		t.Fatal("Load succeeded; Dup is declared in both a.gsx and b.gsx")
	}
	if !errors.Is(err, gsxmail.ErrDuplicateTemplate) {
		t.Errorf("errors.Is(err, ErrDuplicateTemplate) = false; err = %v", err)
	}
}

// TestErrUnknownTemplate is polish item 6's own proof for
// ErrUnknownTemplate: Render given a name Load never found.
func TestErrUnknownTemplate(t *testing.T) {
	set := loadInviteSet(t)
	_, err := set.Render("NoSuchTemplate", InviteProps{})
	if err == nil {
		t.Fatal("Render succeeded; NoSuchTemplate does not exist")
	}
	if !errors.Is(err, gsxmail.ErrUnknownTemplate) {
		t.Errorf("errors.Is(err, ErrUnknownTemplate) = false; err = %v", err)
	}
}

// wrongProps is deliberately not InviteProps, for TestErrPropsMismatch.
type wrongProps struct {
	Something string
}

// TestErrPropsMismatch is polish item 6's own proof for ErrPropsMismatch:
// props is a named struct, but not the template's declared one.
func TestErrPropsMismatch(t *testing.T) {
	set := loadInviteSet(t)
	_, err := set.Render("InviteEmail", wrongProps{Something: "x"})
	if err == nil {
		t.Fatal("Render succeeded; wrongProps is not InviteProps")
	}
	if !errors.Is(err, gsxmail.ErrPropsMismatch) {
		t.Errorf("errors.Is(err, ErrPropsMismatch) = false; err = %v", err)
	}
}

// TestErrNilProps is polish item 6's own proof for ErrNilProps: props is
// a nil pointer to the template's declared props type.
func TestErrNilProps(t *testing.T) {
	set := loadInviteSet(t)
	var props *InviteProps
	_, err := set.Render("InviteEmail", props)
	if err == nil {
		t.Fatal("Render succeeded; props is a nil pointer")
	}
	if !errors.Is(err, gsxmail.ErrNilProps) {
		t.Errorf("errors.Is(err, ErrNilProps) = false; err = %v", err)
	}
}

// TestErrResolve is polish item 6's own proof for ErrResolve: a
// map[string]any props value that doc.Resolve itself rejects — neither a
// struct nor a map is a render-time-only failure Load-time checking
// cannot always reach (a map[string]any has no named Go type to check).
func TestErrResolve(t *testing.T) {
	set := loadInviteSet(t)
	_, err := set.Render("InviteEmail", 42)
	if err == nil {
		t.Fatal("Render succeeded; 42 is neither a struct nor a map[string]any")
	}
	if !errors.Is(err, gsxmail.ErrResolve) {
		t.Errorf("errors.Is(err, ErrResolve) = false; err = %v", err)
	}
}

// TestErrLower is polish item 6's own proof for ErrLower: a template
// that clears the email lint but that Lower still rejects — its root is
// not <email.Shell>.
func TestErrLower(t *testing.T) {
	fsys := fstest.MapFS{
		"badroot.gsx": {Data: []byte(`package emails

func BadRoot() Node {
    return <div>not a Shell root</div>
}
`)},
	}
	_, err := gsxmail.Load(fsys, gsxmail.Options{})
	if err == nil {
		t.Fatal("Load succeeded; BadRoot's root is not <email.Shell>")
	}
	if !errors.Is(err, gsxmail.ErrLower) {
		t.Errorf("errors.Is(err, ErrLower) = false; err = %v", err)
	}
}
