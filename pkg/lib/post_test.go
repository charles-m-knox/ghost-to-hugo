package ghosttohugo_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	ghosttohugo "git.cmcode.dev/cmcode/ghost-to-hugo/pkg/lib"
)

// `<p>Test</p><figure class="foo"><img height="900" width="200" src="foo.png"></figure><p>Test 2</p>`
// `<p>Test</p><figure class="foo"><img height="" width="" src="foo.png"></img></figure><p>Test 2</p>`

func TestProcessGhostPost(t *testing.T) {
	t.Parallel()

	tp := ghosttohugo.GhostPost{
		SqlCreatedAt:   "2006-01-02 15:04:05",
		SqlUpdatedAt:   sql.NullString{String: "2006-01-02 15:04:05", Valid: true},
		SqlPublishedAt: sql.NullString{String: "2006-01-02 15:04:05", Valid: true},
		Status:         ghosttohugo.GhostPostStatusDraft,
	}

	wtp := tp
	wtp.CreatedAt = time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	wtp.UpdatedAt = time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	wtp.PublishedAt = time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)

	badDate := "0-99-99 50:99:99"

	badtpCreatedAt := tp
	badtpCreatedAt.SqlCreatedAt = badDate

	badtpUpdatedAt := tp
	badtpUpdatedAt.SqlUpdatedAt = sql.NullString{String: badDate, Valid: true}

	badtpPublishedAt := tp
	badtpPublishedAt.SqlPublishedAt = sql.NullString{String: badDate, Valid: true}

	tests := []struct {
		c    ghosttohugo.Config
		p    ghosttohugo.GhostPost
		want ghosttohugo.GhostPost
		err  bool
	}{
		{
			ghosttohugo.Config{PublishDrafts: true},
			tp,
			wtp,
			false,
		},
		{
			ghosttohugo.Config{PublishDrafts: true},
			tp,
			wtp,
			false,
		},
		{
			ghosttohugo.Config{PublishDrafts: true},
			badtpCreatedAt,
			badtpCreatedAt,
			true,
		},
		{
			ghosttohugo.Config{PublishDrafts: true},
			badtpUpdatedAt,
			badtpUpdatedAt,
			true,
		},
		{
			ghosttohugo.Config{PublishDrafts: true},
			badtpPublishedAt,
			badtpPublishedAt,
			true,
		},
	}

	for i, test := range tests {
		got, err := test.c.ProcessGhostPost(test.p)
		if err != nil && !test.err {
			t.Logf("test %v failed: got error but didn't expect one: %v", i, err.Error())
			t.FailNow()
		} else if err == nil && test.err {
			t.Logf("test %v failed: didn't get an error but was expecting one", i)
			t.FailNow()
		} else if test.err {
			continue // success - no need to compare results below
		}

		if got.IsDraft != test.want.IsDraft {
			t.Logf("test %v failed: got IsDraft=%v, want %v", i, got.IsDraft, test.want.IsDraft)
			t.Fail()
		}

		if got.CreatedAt != test.want.CreatedAt {
			t.Logf("test %v failed: got CreatedAt=%v, want %v", i, got.CreatedAt, test.want.CreatedAt)
			t.Fail()
		}

		if got.UpdatedAt != test.want.UpdatedAt {
			t.Logf("test %v failed: got UpdatedAt=%v, want %v", i, got.UpdatedAt, test.want.UpdatedAt)
			t.Fail()
		}

		if got.PublishedAt != test.want.PublishedAt {
			t.Logf("test %v failed: got PublishedAt=%v, want %v", i, got.PublishedAt, test.want.PublishedAt)
			t.Fail()
		}
	}

	// special case: test the publish override mechanism - the post should be set
	// to time.Now() - 1 second when overridden
	{
		minuteAgo := time.Now().Add(-1 * time.Minute)
		tc := ghosttohugo.Config{
			SetUnpublishedToNow: true,
		}
		badtpPublishOverride := tp
		badtpPublishOverride.SqlPublishedAt = sql.NullString{String: "", Valid: false}
		got, err := tc.ProcessGhostPost(badtpPublishOverride)
		if err != nil {
			t.Logf("publish override test failed: got error but wasn't expecting one: %v", err.Error())
			t.FailNow()
		}

		if !got.PublishedAt.After(minuteAgo) {
			t.Logf("publish override test failed: time wasn't ahead of the recorded time, got %v", got.PublishedAt)
			t.FailNow()
		}
	}
}

func TestRenderString(t *testing.T) {
	t.Parallel()

	publishTime := time.Now().Add(-1 * time.Hour)

	testConf := ghosttohugo.Config{
		PostTypes:        map[string]bool{},
		PostVisibilities: map[string]bool{},
		PostStatuses:     map[string]bool{},
		FrontMatter: ghosttohugo.FrontMatterConfig{
			Title: ghosttohugo.DefaultFrontMatterTitle,
			Date:  ghosttohugo.DefaultFrontMatterDate,
			Draft: ghosttohugo.DefaultFrontMatterDraft,
			Slug:  ghosttohugo.DefaultFrontMatterSlug,
		},
		ForbidEmptyPosts:  true,
		RawShortcodeStart: ghosttohugo.DefaultRawShortcodeStart,
		RawShortcodeEnd:   ghosttohugo.DefaultRawShortcodeEnd,
		Template:          ghosttohugo.DefaultTemplate,
		GhostURL:          "https://example.com",
	}

	testPost := ghosttohugo.GhostPost{
		HTML: sql.NullString{
			String: `<p>Test</p><figure class="foo"><img height="900" width="200" src="__GHOST_URL__/foo.png"></figure><p>Test 2</p>`,
			Valid:  true,
		},
		Title:       "Test Post",
		Slug:        "test-post",
		PublishedAt: publishTime,
	}

	invalidTestPost := testPost
	invalidTestPost.HTML.Valid = false
	invalidTestPost.HTML.String = "<foo<bar><baz>///></></.>"

	tests := []struct {
		c           ghosttohugo.Config
		p           ghosttohugo.GhostPost
		want        string
		templateErr bool
		renderErr   bool
	}{
		{
			testConf,
			testPost,
			fmt.Sprintf(`---
title: |
  Test Post
date: "%v"
draft: false
slug: test-post
isPost: true
---

{{< rawhtml >}}
<p>Test</p><figure class="foo"><img height="" width="" src="https://example.com/foo.png"></img></figure><p>Test 2</p>
{{</ rawhtml >}}
`, publishTime.Format(time.RFC3339)),
			false,
			false,
		},
		{
			// test getting empty/null html post from db, while forbidding it
			ghosttohugo.Config{ForbidEmptyPosts: true},
			ghosttohugo.GhostPost{HTML: sql.NullString{Valid: false}},
			"",
			false,
			true,
		},
		{
			// test two branchse: a post's HTML from DB impossibly marked as
			// invalid (null from db), but the code continues anyways - later
			// encountering an error during template render
			ghosttohugo.Config{ForbidEmptyPosts: false},
			invalidTestPost,
			"",
			false,
			true,
		},
		{
			// test providing an invalid template
			ghosttohugo.Config{Template: "{{ .InvalidTemplate "},
			invalidTestPost,
			"",
			true,
			false,
		},
	}

	for i, test := range tests {
		err := test.c.ParseTemplate()
		if err != nil && !test.templateErr {
			t.Logf("failed to parse template: %v", err.Error())
			t.FailNow()
		} else if err == nil && test.templateErr {
			t.Logf("test %v did not encounter a template error but wanted one", i)
			t.FailNow()
		} else if err != nil && test.templateErr {
			continue // success - no need to compare the remaining results
		}

		got, err := test.c.RenderString(test.p)
		if err != nil && !test.renderErr {
			t.Logf("test %v failed: got error but didn't expect: %v", i, err.Error())
			t.Fail()
		} else if err == nil && test.renderErr {
			t.Logf("test %v did not encounter a render error but wanted one: %v", i, got)
			t.FailNow()
		}

		if got != test.want {
			t.Logf("test %v failed: got %v, want %v", i, got, test.want)
			t.Fail()
		}
	}
}

func TestIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		c    ghosttohugo.Config
		p    ghosttohugo.GhostPost
		want bool
	}{
		{
			// tests the first if statement
			ghosttohugo.Config{
				PostTypes:        map[string]bool{},
				PostVisibilities: map[string]bool{},
				PostStatuses:     map[string]bool{},
			},
			ghosttohugo.GhostPost{},
			false,
		},
		{
			// tests the first if statement
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": true},
				PostVisibilities: map[string]bool{},
				PostStatuses:     map[string]bool{},
			},
			ghosttohugo.GhostPost{},
			false,
		},
		{
			// tests the first if statement
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": true},
				PostVisibilities: map[string]bool{"bar": true},
				PostStatuses:     map[string]bool{},
			},
			ghosttohugo.GhostPost{},
			false,
		},
		{
			// tests the two branches on checking allowable post types
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": false},
				PostVisibilities: map[string]bool{"bar": true},
				PostStatuses:     map[string]bool{"baz": true},
			},
			ghosttohugo.GhostPost{
				Type: "foo",
			},
			false,
		},
		{
			// tests the two branches on checking allowable post types
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": true},
				PostVisibilities: map[string]bool{"bar": true},
				PostStatuses:     map[string]bool{"baz": true},
			},
			ghosttohugo.GhostPost{
				Type: "foo",
			},
			true,
		},
		{
			// tests the two branches on checking allowable post visibilities
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": true},
				PostVisibilities: map[string]bool{"bar": false},
				PostStatuses:     map[string]bool{"baz": true},
			},
			ghosttohugo.GhostPost{
				Visibility: "bar",
			},
			false,
		},
		{
			// tests the two branches on checking allowable post visibilities
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": true},
				PostVisibilities: map[string]bool{"bar": true},
				PostStatuses:     map[string]bool{"baz": true},
			},
			ghosttohugo.GhostPost{
				Visibility: "bar",
			},
			true,
		},
		{
			// tests the two branches on checking allowable post statuses
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": true},
				PostVisibilities: map[string]bool{"bar": false},
				PostStatuses:     map[string]bool{"baz": false},
			},
			ghosttohugo.GhostPost{
				Status: "baz",
			},
			false,
		},
		{
			// tests the two branches on checking allowable post statuses
			ghosttohugo.Config{
				PostTypes:        map[string]bool{"foo": true},
				PostVisibilities: map[string]bool{"bar": true},
				PostStatuses:     map[string]bool{"baz": true},
			},
			ghosttohugo.GhostPost{
				Status: "baz",
			},
			true,
		},
	}

	for i, test := range tests {
		got := test.c.IsValid(test.p)
		if got != test.want {
			t.Logf("test %v failed: got %v, want %v", i, got, test.want)
			t.Fail()
		}
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	fail := func(s string) {
		t.Log(s)
		t.Fail()
	}

	c := ghosttohugo.Config{
		PostStatuses:     make(map[string]bool),
		PostTypes:        make(map[string]bool),
		PostVisibilities: make(map[string]bool),
		FrontMatter:      ghosttohugo.FrontMatterConfig{},
	}

	c.ApplyDefaults()
	c.Process()

	if c.Template != ghosttohugo.DefaultTemplate {
		fail("c.Template mismatch")
	}

	if c.RawShortcodeStart != ghosttohugo.DefaultRawShortcodeStart {
		fail("c.RawShortcodeStart mismatch")
	}

	if c.RawShortcodeEnd != ghosttohugo.DefaultRawShortcodeEnd {
		fail("c.RawShortcodeEnd mismatch")
	}

	if c.FrontMatter.Title != ghosttohugo.DefaultFrontMatterTitle {
		fail("c.FrontMatter.Title mismatch")
	}

	if c.FrontMatter.Date != ghosttohugo.DefaultFrontMatterDate {
		fail("c.FrontMatter.Date mismatch")
	}

	if c.FrontMatter.Slug != ghosttohugo.DefaultFrontMatterSlug {
		fail("c.FrontMatter.Slug mismatch")
	}

	if c.FrontMatter.Draft != ghosttohugo.DefaultFrontMatterDraft {
		fail("c.FrontMatter.Draft mismatch")
	}

	if len(c.PostStatuses) != 2 {
		fail("len(c.PostStatuses) mismatch")
	}

	if len(c.PostTypes) != 2 {
		fail("len(c.PostTypes) mismatch")
	}

	if len(c.PostVisibilities) != 1 {
		fail("len(c.PostVisibilities) mismatch")
	}

	for k, v := range c.PostStatuses {
		switch k {
		case "published":
			if !v {
				fail("PostStatus published mismatch")
			}
			continue
		case "draft":
			if v {
				fail("PostStatus draft mismatch")
			}
			continue
		}

		fail("PostStatus unexpected key")
	}

	for k, v := range c.PostTypes {
		switch k {
		case "post":
			if !v {
				fail("PostTypes post mismatch")
			}
			continue
		case "page":
			if !v {
				fail("PostTypes page mismatch")
			}
			continue
		}

		fail("PostTypes unexpected key")
	}

	for k, v := range c.PostVisibilities {
		switch k {
		case "public":
			if !v {
				fail("PostVisibilities public mismatch")
			}
			continue
		}

		fail("PostTypes unexpected key")
	}

	if c.LinkReplacements == nil {
		fail("c.LinkReplacements should not be nil")
	}

	if c.ReplaceLinks {
		fail("c.ReplaceLinks should be false")
	}
}
