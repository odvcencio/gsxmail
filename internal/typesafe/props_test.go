package typesafe

import (
	"testing"
	"testing/fstest"
)

func TestResolverResolve(t *testing.T) {
	fsys := fstest.MapFS{
		"emails/props.go": &fstest.MapFile{Data: []byte(`package emails

type InviteProps struct {
	ShortDate string
	Count     int
	Rate      float64
	Sent      bool
	Tags      []string
}
`)},
	}
	r := NewResolver(fsys)

	props, err := r.Resolve("emails", "InviteProps")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if props.Name != "InviteProps" {
		t.Errorf("Name = %q, want InviteProps", props.Name)
	}
	want := map[string]Kind{
		"ShortDate": KindString,
		"Count":     KindInt,
		"Rate":      KindFloat,
		"Sent":      KindBool,
		"Tags":      KindSlice,
	}
	for name, kind := range want {
		f, ok := props.Fields[name]
		if !ok {
			t.Errorf("field %s not resolved", name)
			continue
		}
		if f.Kind != kind {
			t.Errorf("field %s Kind = %v, want %v", name, f.Kind, kind)
		}
	}
	if got := props.Fields["Tags"].ElemKind; got != KindString {
		t.Errorf("Tags ElemKind = %v, want KindString", got)
	}
}

func TestResolverResolveEmptyTypeName(t *testing.T) {
	fsys := fstest.MapFS{"emails/props.go": &fstest.MapFile{Data: []byte("package emails\n")}}
	r := NewResolver(fsys)
	props, err := r.Resolve("emails", "")
	if err != nil || props != nil {
		t.Fatalf("Resolve(dir, \"\") = %v, %v; want nil, nil", props, err)
	}
}

func TestResolverResolvePointerType(t *testing.T) {
	fsys := fstest.MapFS{
		"emails/props.go": &fstest.MapFile{Data: []byte(`package emails

type P struct {
	Name string
}
`)},
	}
	r := NewResolver(fsys)
	props, err := r.Resolve("emails", "*P")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if props.Name != "P" {
		t.Errorf("Name = %q, want P (the leading * must be stripped)", props.Name)
	}
}

func TestBareFieldName(t *testing.T) {
	field, ok := BareFieldName("props.ShortDate", "props")
	if !ok || field != "ShortDate" {
		t.Errorf("BareFieldName(props.ShortDate) = %q, %v; want ShortDate, true", field, ok)
	}
	if _, ok := BareFieldName(`"a" + props.ShortDate`, "props"); ok {
		t.Error("BareFieldName should reject a non-bare expression")
	}
}
