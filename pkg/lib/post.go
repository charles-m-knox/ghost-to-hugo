package ghosttohugo

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
	"time"
)

// FrontMatterConfig determines what strings to use for various front matter
// values - for example, if FrontMatterConfig.Title is set to a value of
// "pageTitle", the front matter will be rendered like this:
//
//	`
//	---
//	pageTitle: {{ Post.Title }}
//	# ...
//	---
//	`
type FrontMatterConfig struct {
	// The string to use instead of 'title' in front matter.
	Title string `json:"title"`
	// The string to use instead of 'date' in front matter.
	Date string `json:"date"`
	// The string to use instead of 'draft' in front matter.
	Draft string `json:"draft"`
	// https://gohugo.io/content-management/urls/#slug
	Slug string `json:"slug"`
}

type Config struct {
	// Connection string for the mysql database.
	MySQLConnectionString string `json:"mysqlConnectionString"`
	// Values to use for the front matter.
	FrontMatter FrontMatterConfig `json:"frontMatter"`
	// Your theme's shortcode that starts the output of raw html, such as:
	// 	"{{< rawhtml >}}"
	RawShortcodeStart string `json:"rawShortcodeStart"`
	// Your theme's shortcode that ends the output of raw html, such as:
	// 	"{{</ rawHtml >}}"
	RawShortcodeEnd string `json:"rawShortcodeEnd"`
	// Path to save rendered markdown files to.
	OutputPath string `json:"outputPath"`
	// All occurrences of __GHOST_URL__ will be replaced with this string - this
	// is required in order to make images work, as well as other things.
	GhostURL string `json:"ghostUrl"`
	// Values are typically "post": true or "page": true.
	PostTypes map[string]bool `json:"postTypes"`
	// Values are typically "published": true, "draft": false.
	// Depending on your setup, you may set "sent": false.
	PostStatuses map[string]bool `json:"postStatuses"`
	// Values are typically "public": true.
	PostVisibilities map[string]bool `json:"postVisibilities"`
	// If true, empty (null) posts will cause the program to halt.
	ForbidEmptyPosts bool `json:"forbidEmptyPosts"`
	// If true, posts without publication dates with be set to now.
	SetUnpublishedToNow bool `json:"setUnpublishedToNow"`
	// If true, all posts will not be marked as drafts.
	PublishDrafts bool `json:"publishDrafts"`
	// A mapping of strings to replace within <a href=""> tags. This will only
	// be done if ReplaceLinks is set to true (which will under normal
	// circumstances be determined if a non-zero length map is provided for
	// LinkReplacements, but if you are doing something abnormal, you should set
	// ReplaceLinks to true manually.)
	LinkReplacements map[string]string `json:"linkReplacements"`
	// The template that will be rendered.
	//
	// The front matter will be placed at the top of every page. Usage looks
	// like this:
	//
	//  `
	//  ---
	//  {{ .FrontMatterConfig.Title }}: {{ .Post.Title }}
	//  {{ .FrontMatterConfig.Date }}: "{{ .Post.PublishedAt }}"
	//  {{ .FrontMatterConfig.Draft }}: {{ .Post.IsDraft }}
	//  customProperty: customValue # etc.
	//  ---
	//
	//  {{ .PostHtml }}
	//  `
	Template string `json:"template"`

	// Parsed template - parsed once, reused later many times.
	template *template.Template

	// If SetUnpublishedToNow is set to true, the last-used time is stored here.
	// Each post's publication time is decremented by 1 second.
	lastPublishOverride time.Time

	// If true, <a href=""> tags will have each key replaced with its respective
	// value. This is useful for swapping things like http://example.com with
	// https://nojs.example.com.
	//
	// Typically this gets set to true if using the
	// expected workflow for this program, but if you are doing something
	// out of the ordinary, please set this to true and then use the
	// LinkReplacements map to define link replacements.
	ReplaceLinks bool
}

// GhostPost is the data type that exists in the mysql database as of Ghost
// v5.87.1.
type GhostPost struct {
	ID                       string
	UUID                     string
	Title                    string
	Slug                     string
	Mobiledoc                sql.NullString
	Lexical                  sql.NullString
	HTML                     sql.NullString
	CommentID                sql.NullString
	Plaintext                sql.NullString
	FeatureImage             sql.NullString
	Featured                 bool
	Type                     string
	Status                   string
	Locale                   sql.NullString
	Visibility               string
	EmailRecipientFilter     string
	CreatedAt                time.Time
	CreatedBy                string
	UpdatedAt                time.Time // sql.NullTime
	UpdatedBy                sql.NullString
	PublishedAt              time.Time // sql.NullTime
	PublishedBy              sql.NullString
	CustomExcerpt            sql.NullString
	CodeinjectionHead        sql.NullString
	CodeinjectionFoot        sql.NullString
	CustomTemplate           sql.NullString
	CanonicalUrl             sql.NullString
	NewsletterId             sql.NullString
	ShowTitleAndFeatureImage bool

	// On the Go side, we need to parse these values before putting them into
	// the [GhostPost] struct.
	SqlCreatedAt string // time.Time
	// On the Go side, we need to parse these values before putting them into
	// the [GhostPost] struct.
	SqlUpdatedAt sql.NullString // sql.NullTime
	// On the Go side, we need to parse these values before putting them into
	// the [GhostPost] struct.
	SqlPublishedAt sql.NullString // sql.NullTime

	// This module parses the status field and determines if it's a draft. This
	// isn't specifically a boolean value in the database. It may not always
	// be accurate if Ghost has special logic that determines if something is
	// a draft or not.
	IsDraft bool
}

// All of the fields that map to [GhostPost].
const QUERY_POSTS_FIELDS = `
id as ID,
uuid as UUID,
title as Title,
slug as Slug,
mobiledoc as Mobiledoc,
lexical as Lexical,
html as HTML,
comment_id as CommentID,
plaintext as Plaintext,
feature_image as FeatureImage,
featured as Featured,
type as Type,
status as Status,
locale as Locale,
visibility as Visibility,
email_recipient_filter as EmailRecipientFilter,
created_at as CreatedAt,
created_by as CreatedBy,
updated_at as UpdatedAt,
updated_by as UpdatedBy,
published_at as PublishedAt,
published_by as PublishedBy,
custom_excerpt as CustomExcerpt,
codeinjection_head as CodeinjectionHead,
codeinjection_foot as CodeinjectionFoot,
custom_template as CustomTemplate,
canonical_url as CanonicalUrl,
newsletter_id as NewsletterId,
show_title_and_feature_image as ShowTitleAndFeatureImage
`

const GhostPostStatusDraft = "draft"

// ProcessGhostPost is called by [GetGhostPost] and fills in/processes fields
// that are required in order for this module to fulfill its intended purpose.
func (c *Config) ProcessGhostPost(post GhostPost) (GhostPost, error) {
	var err error

	post.CreatedAt, err = time.Parse("2006-01-02 15:04:05", post.SqlCreatedAt)
	if err != nil {
		return post, fmt.Errorf("failed to parse CreatedAt datetime: %v", err.Error())
	}

	if post.SqlUpdatedAt.Valid {
		post.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", post.SqlUpdatedAt.String)
		if err != nil {
			return post, fmt.Errorf("failed to parse UpdatedAt datetime: %v", err.Error())
		}
	}

	if post.SqlPublishedAt.Valid {
		post.PublishedAt, err = time.Parse("2006-01-02 15:04:05", post.SqlPublishedAt.String)
		if err != nil {
			return post, fmt.Errorf("failed to parse PublishedAt datetime: %v", err.Error())
		}
	} else if !post.SqlPublishedAt.Valid && c.SetUnpublishedToNow {
		if c.lastPublishOverride.IsZero() {
			c.lastPublishOverride = time.Now()
		}
		c.lastPublishOverride = c.lastPublishOverride.Add(time.Duration(-1 * time.Second))
		post.PublishedAt = c.lastPublishOverride
	}

	// determine if it is a draft or not (needed later)
	if post.Status == GhostPostStatusDraft {
		post.IsDraft = true
	}

	if c.PublishDrafts {
		post.IsDraft = false
	}

	return post, nil
}

// GetGhostPost parses an SQL row-yielding iterator and returns a [GhostPost]
// from it.
//
// Usage:
//
// rows, err := db.Query(fmt.Sprintf("SELECT %v FROM posts LIMIT 10", g2h.QUERY_POSTS_FIELDS))
//
//	if err != nil {
//		log.Fatalf("failed to query posts from db: %v", err.Error())
//	}
//
//	defer rows.Close()
//
//	for rows.Next() {
//			post, err := ghosttohugo.GetGhostPost(rows)
//			if err != nil {
//			// ...
//		}
//	}
func (c *Config) GetGhostPost(rows *sql.Rows) (GhostPost, error) {
	var post GhostPost

	err := rows.Scan(&post.ID, &post.UUID, &post.Title, &post.Slug, &post.Mobiledoc, &post.Lexical, &post.HTML, &post.CommentID, &post.Plaintext, &post.FeatureImage, &post.Featured, &post.Type, &post.Status, &post.Locale, &post.Visibility, &post.EmailRecipientFilter, &post.SqlCreatedAt, &post.CreatedBy, &post.SqlUpdatedAt, &post.UpdatedBy, &post.SqlPublishedAt, &post.PublishedBy, &post.CustomExcerpt, &post.CodeinjectionHead, &post.CodeinjectionFoot, &post.CustomTemplate, &post.CanonicalUrl, &post.NewsletterId, &post.ShowTitleAndFeatureImage)
	if err != nil {
		return post, fmt.Errorf("failed to marshal row into interface: %v", err.Error())
	}

	return c.ProcessGhostPost(post)
}

const parsedTemplateName = "template"

// ParseTemplate parses the user-configured template. This should only need to
// be run once.
func (conf *Config) ParseTemplate() error {
	var err error

	conf.template, err = template.New(parsedTemplateName).Parse(conf.Template)
	if err != nil {
		return fmt.Errorf("failed to parse user-configured template: %w", err)
	}

	return nil
}

type postTemplate struct {
	FrontMatterConfig FrontMatterConfig
	Post              GhostPost
	PostDate          string // Post's date rendered as a standard string
	PostHTML          string
	RawShortcodeStart string
	RawShortcodeEnd   string
}

const ghostUrl = "__GHOST_URL__"

// Renders a Ghost post to Hugo markdown.
func (c *Config) RenderString(post GhostPost) (string, error) {
	if !post.HTML.Valid {
		if c.ForbidEmptyPosts {
			return "", fmt.Errorf("post %v html is null, cannot render", post.ID)
		}

		// return "", nil
	}

	h := strings.ReplaceAll(post.HTML.String, ghostUrl, c.GhostURL)

	var err error
	h, err = c.ProcessHTML(h)
	if err != nil {
		return "", fmt.Errorf("failed to process html: %w", err)
	}

	b := bytes.NewBuffer([]byte{})
	err = c.template.Execute(b, postTemplate{
		FrontMatterConfig: c.FrontMatter,
		Post:              post,
		PostDate:          post.PublishedAt.Format(time.RFC3339),
		PostHTML:          h,
		RawShortcodeStart: c.RawShortcodeStart,
		RawShortcodeEnd:   c.RawShortcodeEnd,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render post: %w", err)
	}

	return b.String(), nil
}

// Default values used in the config if not set.
const (
	DefaultRawShortcodeStart = "{{< rawhtml >}}"
	DefaultRawShortcodeEnd   = "{{</ rawhtml >}}"
	DefaultTemplate          = `---
{{ .FrontMatterConfig.Title }}: |
  {{ .Post.Title }}
{{ .FrontMatterConfig.Date }}: "{{ .PostDate }}"
{{ .FrontMatterConfig.Draft }}: {{ .Post.IsDraft }}
{{ .FrontMatterConfig.Slug }}: {{ .Post.Slug }}
isPost: true
---

{{ .RawShortcodeStart }}
{{ .PostHTML }}
{{ .RawShortcodeEnd }}
`
)

// ApplyDefaults applies sensible defaults to the config if left unconfigured.
// You shouldn't normally need to execute this, because it's called
// automatically by [LoadConfig].
func (c *Config) ApplyDefaults() {
	if c.Template == "" {
		c.Template = DefaultTemplate
	}

	// warning: if you change any of these, please update the unit tests!

	if len(c.PostStatuses) == 0 {
		c.PostStatuses = map[string]bool{"published": true, "draft": false}
	}

	if len(c.PostTypes) == 0 {
		c.PostTypes = map[string]bool{"post": true, "page": true}
	}

	if len(c.PostVisibilities) == 0 {
		c.PostVisibilities = map[string]bool{"public": true}
	}

	if c.RawShortcodeStart == "" {
		c.RawShortcodeStart = DefaultRawShortcodeStart
	}

	if c.RawShortcodeEnd == "" {
		c.RawShortcodeEnd = DefaultRawShortcodeEnd
	}

	c.FrontMatter.ApplyDefaults()
}

// Default value that is used for front matter in lieu of a user-configured
// value.
const (
	DefaultFrontMatterTitle = "title"
	DefaultFrontMatterDate  = "date"
	DefaultFrontMatterSlug  = "slug"
	DefaultFrontMatterDraft = "draft"
)

// ApplyDefaults applies sensible defaults to the front matter config if left
// unconfigured. You shouldn't normally need to execute this, because it's
// called automatically by [LoadConfig].
func (f *FrontMatterConfig) ApplyDefaults() {
	if f.Title == "" {
		f.Title = DefaultFrontMatterTitle
	}
	if f.Date == "" {
		f.Date = DefaultFrontMatterDate
	}
	if f.Draft == "" {
		f.Draft = DefaultFrontMatterDraft
	}
	if f.Slug == "" {
		f.Slug = DefaultFrontMatterSlug
	}
}

// makeOutputDir ensures that the desired output directory exists.
func (c *Config) makeOutputDir() error {
	if c.OutputPath == "" {
		return nil
	}

	err := os.MkdirAll(c.OutputPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to make output dir %v: %w", c.OutputPath, err)
	}

	return nil
}

// Process makes a few changes to the config based on the user-provided
// configuration. You shouldn't normally need to execute this, because it's
// called automatically by [LoadConfig].
func (c *Config) Process() {
	if c.LinkReplacements == nil {
		c.LinkReplacements = make(map[string]string)
	}

	c.ReplaceLinks = len(c.LinkReplacements) > 0
}

// LoadConfig reads from file f and applies sensible defaults to values not
// specifically set by the user.
func LoadConfig(f string) (Config, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load config from %v: %v", f, err)
	}

	var c Config
	err = json.Unmarshal(b, &c)
	if err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config from %v: %v", f, err)
	}

	c.ApplyDefaults()
	c.Process()

	err = c.makeOutputDir()
	if err != nil {
		return c, fmt.Errorf("failed to make output dir when loading config: %w", err)
	}

	err = c.ParseTemplate()
	if err != nil {
		return c, fmt.Errorf("failed to parse template when loading config: %w", err)
	}

	return c, nil
}

// Renders all the markdown posts from Ghost to the target directory.
func (c *Config) RenderAll(p []GhostPost) error {
	for _, p := range p {
		_, _, err := c.RenderOne(p)
		if err != nil {
			return fmt.Errorf("failed to render posts: %w", err)
		}
	}

	return nil
}

// Renders all the markdown posts from Ghost to the target directory. Returns
// the number of bytes written and  the full file path that was written to.
func (c *Config) RenderOne(p GhostPost) (int, string, error) {
	b, err := c.RenderString(p)
	if err != nil {
		return 0, "", fmt.Errorf("failed to render post %v: %w", p.UUID, err)
	}

	f := path.Join(c.OutputPath, fmt.Sprintf("%v.md", p.Slug))
	err = os.WriteFile(f, []byte(b), 0o644)
	if err != nil {
		return 0, "", fmt.Errorf("failed to write post %v to %v: %w", p.UUID, f, err)
	}

	return len(b), f, nil
}

// IsValid returns true or false if the post meets the criteria for being
// rendered based on the user-provided configuration. This doesn't get
// called by any of the other Render functions - it's up to you if you want to
// perform any filtering at all.
//
// This function does not do any checks regarding the published/draft state.
func (c *Config) IsValid(p GhostPost) bool {
	if len(c.PostTypes) == 0 || len(c.PostStatuses) == 0 ||
		len(c.PostVisibilities) == 0 {
		return false
	}

	for k, v := range c.PostTypes {
		if p.Type == k && !v {
			return false
		}
	}

	for k, v := range c.PostStatuses {
		if p.Status == k && !v {
			return false
		}
	}

	for k, v := range c.PostVisibilities {
		if p.Visibility == k && !v {
			return false
		}
	}

	return true
}
