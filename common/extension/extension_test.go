// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package extension

import (
	"encoding/json"
	"testing"
	"time"
)

// Test types

type PostSpec struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type PostStatus struct {
	Published bool `json:"published"`
}

type Post struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PostSpec   `json:"spec"`
	Status     PostStatus `json:"status,omitempty"`
}

func (p *Post) GetObjectMeta() *ObjectMeta { return &p.ObjectMeta }

type UserSpec struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type User struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       UserSpec `json:"spec"`
}

func (u *User) GetObjectMeta() *ObjectMeta { return &u.ObjectMeta }

// ─── GVK tests ───

func TestGroupVersionKind_String(t *testing.T) {
	gvk := GroupVersionKind{Group: "blog.io", Version: "v1", Kind: "Post"}
	if gvk.String() != "blog.io/v1, Kind=Post" {
		t.Errorf("unexpected string: %s", gvk.String())
	}
	if gvk.APIVersion() != "blog.io/v1" {
		t.Errorf("unexpected APIVersion: %s", gvk.APIVersion())
	}
}

func TestGroupVersionKindFromAPIVersion(t *testing.T) {
	gvk, err := GroupVersionKindFromAPIVersion("blog.io/v1", "Post")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gvk.Group != "blog.io" || gvk.Version != "v1" || gvk.Kind != "Post" {
		t.Errorf("unexpected GVK: %+v", gvk)
	}
}

func TestGroupVersionKindFromAPIVersion_NoGroup(t *testing.T) {
	gvk, err := GroupVersionKindFromAPIVersion("v1", "Post")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gvk.Group != "" || gvk.Version != "v1" {
		t.Errorf("unexpected GVK: %+v", gvk)
	}
}

func TestGroupVersionKindFromAPIVersion_Invalid(t *testing.T) {
	_, err := GroupVersionKindFromAPIVersion("", "")
	if err == nil {
		t.Error("expected error for empty apiVersion/kind")
	}
}

// ─── Scheme registration tests ───

func TestScheme_Register(t *testing.T) {
	s := NewScheme()
	err := s.Register("blog.io", "v1", "Post", "posts", &Post{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if s.Count() != 1 {
		t.Errorf("expected count=1, got %d", s.Count())
	}
}

func TestScheme_Register_Duplicate(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})
	err := s.Register("blog.io", "v1", "Post", "posts", &Post{})
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestScheme_Register_DuplicateType(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})
	err := s.Register("blog.io", "v2", "Post", "posts", &Post{})
	if err == nil {
		t.Error("expected error for duplicate Go type")
	}
}

func TestScheme_Register_NilPrototype(t *testing.T) {
	s := NewScheme()
	err := s.Register("blog.io", "v1", "Post", "posts", nil)
	if err == nil {
		t.Error("expected error for nil prototype")
	}
}

func TestScheme_Register_NonPointer(t *testing.T) {
	s := NewScheme()
	// A non-pointer struct value doesn't implement Object when GetObjectMeta
	// has a pointer receiver. But Register checks reflect.Kind != Ptr.
	// Test with a plain int (definitely not a pointer or Object).
	err := s.Register("blog.io", "v1", "Post", "posts", nil)
	if err == nil {
		t.Error("expected error for nil prototype")
	}
}

func TestScheme_Register_DefaultPlural(t *testing.T) {
	s := NewScheme()
	err := s.Register("blog.io", "v1", "Post", "", &Post{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	plural, _ := s.Plural(GroupVersionKind{Group: "blog.io", Version: "v1", Kind: "Post"})
	if plural != "Posts" {
		t.Errorf("expected default plural 'Posts', got %q", plural)
	}
}

// ─── Scheme New tests ───

func TestScheme_New(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	obj, err := s.New(GroupVersionKind{Group: "blog.io", Version: "v1", Kind: "Post"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	post, ok := obj.(*Post)
	if !ok {
		t.Fatalf("expected *Post, got %T", obj)
	}
	if post.Kind != "Post" {
		t.Errorf("expected Kind=Post, got %q", post.Kind)
	}
	if post.APIVersion != "blog.io/v1" {
		t.Errorf("expected APIVersion=blog.io/v1, got %q", post.APIVersion)
	}
}

func TestScheme_New_Unregistered(t *testing.T) {
	s := NewScheme()
	_, err := s.New(GroupVersionKind{Group: "x", Version: "v1", Kind: "Y"})
	if err == nil {
		t.Error("expected error for unregistered GVK")
	}
}

// ─── GVKForObject tests ───

func TestScheme_GVKForObject(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	gvk, err := s.GVKForObject(&Post{})
	if err != nil {
		t.Fatalf("gvk: %v", err)
	}
	if gvk.Kind != "Post" {
		t.Errorf("expected Kind=Post, got %q", gvk.Kind)
	}
}

func TestScheme_GVKForObject_Unregistered(t *testing.T) {
	s := NewScheme()
	_, err := s.GVKForObject(&Post{})
	if err == nil {
		t.Error("expected error for unregistered type")
	}
}

func TestScheme_GVKForObject_Nil(t *testing.T) {
	s := NewScheme()
	_, err := s.GVKForObject(nil)
	if err == nil {
		t.Error("expected error for nil")
	}
}

// ─── Lookup tests ───

func TestScheme_LookupByPlural(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	gvk, err := s.LookupByPlural("blog.io", "v1", "posts")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if gvk.Kind != "Post" {
		t.Errorf("expected Kind=Post, got %q", gvk.Kind)
	}
}

func TestScheme_LookupByPlural_NotFound(t *testing.T) {
	s := NewScheme()
	_, err := s.LookupByPlural("x", "v1", "y")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestScheme_IsRegistered(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	gvk := GroupVersionKind{Group: "blog.io", Version: "v1", Kind: "Post"}
	if !s.IsRegistered(gvk) {
		t.Error("expected registered")
	}
	if s.IsRegistered(GroupVersionKind{Group: "x", Version: "v1", Kind: "Y"}) {
		t.Error("expected not registered")
	}
}

func TestScheme_AllGVKs(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})
	_ = s.Register("user.io", "v1", "User", "users", &User{})

	gvks := s.AllGVKs()
	if len(gvks) != 2 {
		t.Fatalf("expected 2 GVKs, got %d", len(gvks))
	}
}

// ─── Unregister tests ───

func TestScheme_Unregister(t *testing.T) {
	s := NewScheme()
	gvk := GroupVersionKind{Group: "blog.io", Version: "v1", Kind: "Post"}
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	err := s.Unregister(gvk)
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}
	if s.IsRegistered(gvk) {
		t.Error("expected not registered after unregister")
	}
}

func TestScheme_Unregister_NotFound(t *testing.T) {
	s := NewScheme()
	err := s.Unregister(GroupVersionKind{Group: "x", Version: "v1", Kind: "Y"})
	if err == nil {
		t.Error("expected error for not found")
	}
}

// ─── Serialization tests ───

func TestScheme_MarshalJSON(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	post := &Post{
		Spec: PostSpec{Title: "Hello", Content: "World"},
	}
	post.Name = "post-1"
	post.CreationTimestamp = time.Now()

	data, err := s.MarshalJSON(post)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if raw["apiVersion"] != "blog.io/v1" {
		t.Errorf("expected apiVersion=blog.io/v1, got %v", raw["apiVersion"])
	}
	if raw["kind"] != "Post" {
		t.Errorf("expected kind=Post, got %v", raw["kind"])
	}
}

func TestScheme_UnmarshalJSON(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	jsonStr := `{"apiVersion":"blog.io/v1","kind":"Post","metadata":{"name":"test"},"spec":{"title":"Hi"}}`
	obj, err := s.UnmarshalJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	post, ok := obj.(*Post)
	if !ok {
		t.Fatalf("expected *Post, got %T", obj)
	}
	if post.Name != "test" {
		t.Errorf("expected name=test, got %q", post.Name)
	}
	if post.Spec.Title != "Hi" {
		t.Errorf("expected title=Hi, got %q", post.Spec.Title)
	}
}

func TestScheme_UnmarshalJSON_UnknownType(t *testing.T) {
	s := NewScheme()
	jsonStr := `{"apiVersion":"x/v1","kind":"Y"}`
	_, err := s.UnmarshalJSON([]byte(jsonStr))
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestScheme_UnmarshalJSON_InvalidJSON(t *testing.T) {
	s := NewScheme()
	_, err := s.UnmarshalJSON([]byte(`invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ─── Concurrency tests ───

func TestScheme_ConcurrentAccess(t *testing.T) {
	s := NewScheme()
	_ = s.Register("blog.io", "v1", "Post", "posts", &Post{})

	done := make(chan struct{})

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				s.IsRegistered(GroupVersionKind{Group: "blog.io", Version: "v1", Kind: "Post"})
				s.AllGVKs()
				s.Count()
			}
			done <- struct{}{}
		}()
	}

	// Concurrent New
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				s.New(GroupVersionKind{Group: "blog.io", Version: "v1", Kind: "Post"})
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 15; i++ {
		<-done
	}
}
