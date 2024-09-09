package ghosttohugo_test

import (
	"testing"

	ghosttohugo "github.com/charles-m-knox/ghost-to-hugo/pkg/lib"
)

func TestProcessHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		c    ghosttohugo.Config
		s    string
		want string
		err  bool
	}{
		{
			ghosttohugo.Config{},
			`<p>Test</p><img height="900" width="200" src="foo.png"><p>Test 2</p>`,
			`<p>Test</p><img height="" width="" src="foo.png"></img><p>Test 2</p>`,
			false,
		},
		{
			ghosttohugo.Config{
				ReplaceLinks: true,
				LinkReplacements: map[string]string{
					"https://example.com": "https://nojs.example.com",
				},
			},
			`<p>Test</p><img height="900" width="200" src="foo.png"><p><a href="https://example.com" rel="noopener noreferrer nofollow">Example.com</a></p><p>Test 3</p>`,
			`<p>Test</p><img height="" width="" src="foo.png"></img><p><a href="https://nojs.example.com" rel="noopener noreferrer nofollow">Example.com</a></p><p>Test 3</p>`,
			false,
		},
		{
			ghosttohugo.Config{},
			`<p>Test</p><figure class="foo"><img height="900" width="200" src="foo.png"></figure><p>Test 2</p>`,
			`<p>Test</p><figure class="foo"><img height="" width="" src="foo.png"></img></figure><p>Test 2</p>`,
			false,
		},
	}

	for i, test := range tests {
		got, err := test.c.ProcessHTML(test.s)
		if err != nil && !test.err {
			t.Logf("test %v unexpectedly failed: %v", i, err.Error())
			t.Fail()
		}

		if got != test.want {
			t.Logf("test %v failed: got %v, want %v", i, got, test.want)
			t.Fail()
		}
	}
}
